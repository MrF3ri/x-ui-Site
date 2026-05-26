package eventbus

type Event struct {
	Topic   string
	Payload any
}
type Bus struct{ subs map[string][]chan Event }

func New() *Bus { return &Bus{subs: map[string][]chan Event{}} }
func (b *Bus) Subscribe(topic string) <-chan Event {
	ch := make(chan Event, 8)
	b.subs[topic] = append(b.subs[topic], ch)
	return ch
}
func (b *Bus) Publish(e Event) {
	for _, ch := range b.subs[e.Topic] {
		select {
		case ch <- e:
		default:
		}
	}
}
