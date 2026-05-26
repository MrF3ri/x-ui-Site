package provisioning

import (
    "context"
    "database/sql"
    "encoding/json"
    "fmt"
    "log"
    "time"

    "garudapanel/internal/notification"
    "garudapanel/internal/repository"
    "garudapanel/internal/xui"
)

type WorkerV2 struct {
    db       *sql.DB
    jobs     *repository.ProvisioningJobRepository
    orders   *repository.OrderRepository
    catalog  *repository.CatalogRepository
    panels   *repository.XUIPanelRepository
    proxy    *repository.ProxyServiceRepository
    adapter  xui.Adapter
    notifier *notification.Hub
    maxRetries int
}

func NewWorkerV2(db *sql.DB, jobs *repository.ProvisioningJobRepository, orders *repository.OrderRepository, catalog *repository.CatalogRepository, panels *repository.XUIPanelRepository, proxy *repository.ProxyServiceRepository, adapter xui.Adapter, notifier *notification.Hub) *WorkerV2 {
    return &WorkerV2{db: db, jobs: jobs, orders: orders, catalog: catalog, panels: panels, proxy: proxy, adapter: adapter, notifier: notifier, maxRetries: 5}
}

func (w *WorkerV2) Start(ctx context.Context) {
    go func() {
        ticker := time.NewTicker(2 * time.Second)
        defer ticker.Stop()
        for {
            select {
            case <-ctx.Done():
                return
            case <-ticker.C:
                jobID, orderID, vendorID, err := w.jobs.ClaimNext(ctx)
                if err != nil {
                    // no job available or db error
                    continue
                }
                if err := w.handleJob(ctx, jobID, orderID, vendorID); err != nil {
                    log.Printf("{"+"\"level\":\"error\",\"msg\":\"provisioning.job.failed\",\"job_id\":%d,\"order_id\":%d,\"err\":%q}" , jobID, orderID, err)
                    // determine if retries exceeded
                    // fetch current retries would require extra query; optimistic: mark failed with dead when retries exceeded
                    // For simplicity, mark failed and set dead=false to allow Repo to increment retries; if retries>maxRetries, mark dead true
                    // Here we fetch retries
                    var retries int
                    _ = w.db.QueryRow("SELECT retries FROM provisioning_jobs WHERE id=$1", jobID).Scan(&retries)
                    dead := retries+1 >= w.maxRetries
                    _ = w.jobs.MarkFailed(ctx, jobID, err.Error(), dead)
                    continue
                }
                _ = w.jobs.MarkDone(ctx, jobID)
            }
        }
    }()
}

func (w *WorkerV2) handleJob(ctx context.Context, jobID, orderID, vendorID int64) error {
    // load order (vendor enforced)
    o, err := w.orders.ByID(ctx, vendorID, orderID)
    if err != nil { return err }
    // fetch panel
    panel, err := w.panels.FirstActive(ctx, vendorID)
    if err != nil { return err }
    // fetch catalog details
    cat, err := w.catalog.GetByID(o.CatalogID)
    if err != nil { return err }
    // prepare provision request
    uuid := fmt.Sprintf("svc-%d-%d", o.UserID, time.Now().UnixNano())
    req := xui.ProvisionRequest{VendorID: o.VendorID, UserID: o.UserID, Email: "user", UUID: uuid, DurationDays: cat.DurationDays, TrafficGB: cat.TrafficLimitGB, Protocol: cat.Protocol}
    // call adapter (network call)
    pr, err := w.adapter.Provision(xui.Panel{VendorID: panel.VendorID, Name: panel.Name, URL: panel.URL, Token: panel.Token, InboundID: panel.InboundID}, req)
    if err != nil {
        return err
    }
    // Build subscription URL and QR payload (simple)
    subURL := fmt.Sprintf("%s/sub/%s", panel.URL, pr.UUID)
    qrPayload := subURL
    // Persist proxy service and link to order in transaction
    tx, err := w.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
    if err != nil { return err }
    defer func() { if err != nil { _ = tx.Rollback() } }()
    var svcID int64
    svcID, err = w.proxy.CreateTx(ctx, tx, repository.ProxyServiceRecord{
        VendorID: panel.VendorID,
        UserID: o.UserID,
        OrderID: &o.ID,
        PanelID: nil,
        UUID: pr.UUID,
        Protocol: cat.Protocol,
        SubscriptionURL: subURL,
        QRPayload: qrPayload,
        ConfigPayload: "",
        Status: "active",
        ExpiresAt: &pr.ExpiresAt,
        TrafficLimitGB: cat.TrafficLimitGB,
        DurationDays: cat.DurationDays,
    })
    if err != nil { return err }
    // link order → service
    if err = w.orders.LinkServiceTx(ctx, tx, o.ID, svcID); err != nil { return err }
    if err = tx.Commit(); err != nil { return err }
    // notify
    payload, _ := json.Marshal(map[string]any{"order_id": o.ID, "service_id": svcID})
    log.Printf("{"+"\"level\":\"info\",\"msg\":\"provisioning.succeeded\",\"payload\":%q}", string(payload))
    w.notifier.Notify("service.provisioned", map[string]any{"order_id": o.ID, "service_id": svcID})
    return nil
}
