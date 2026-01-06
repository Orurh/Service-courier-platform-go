package kafka

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/IBM/sarama"
	"github.com/stretchr/testify/require"

	"course-go-avito-Orurh/internal/service/orders"
	testlog "course-go-avito-Orurh/internal/testutil"
)

type fakeSession struct {
	ctx context.Context

	mu     sync.Mutex
	marked int
}

func (s *fakeSession) Context() context.Context { return s.ctx }

func (s *fakeSession) MarkMessage(*sarama.ConsumerMessage, string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.marked++
}

func (s *fakeSession) MarkOffset(string, int32, int64, string)  {}
func (s *fakeSession) Commit()                                  {}
func (s *fakeSession) ResetOffset(string, int32, int64, string) {}
func (s *fakeSession) Claims() map[string][]int32               { return nil }
func (s *fakeSession) MemberID() string                         { return "" }
func (s *fakeSession) GenerationID() int32                      { return 0 }

func (s *fakeSession) MarkedCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.marked
}

type fakeClaim struct {
	ch chan *sarama.ConsumerMessage
}

func (c fakeClaim) Topic() string              { return "t" }
func (c fakeClaim) Partition() int32           { return 0 }
func (c fakeClaim) InitialOffset() int64       { return 0 }
func (c fakeClaim) HighWaterMarkOffset() int64 { return 0 }
func (c fakeClaim) Messages() <-chan *sarama.ConsumerMessage {
	return c.ch
}

func TestConsumeClaim_BadJSON_Skips(t *testing.T) {
	t.Parallel()

	rec := testlog.New()
	c := &Consumer{
		logger: rec.Logger(),
		handler: func(context.Context, orders.Event) error {
			t.Fatal("handler must not be called")
			return nil
		},
	}
	h := &groupHandler{c: c}

	sess := &fakeSession{ctx: context.Background()}
	msgCh := make(chan *sarama.ConsumerMessage, 1)
	msgCh <- &sarama.ConsumerMessage{Value: []byte("not-json")}
	close(msgCh)

	err := h.ConsumeClaim(sess, fakeClaim{ch: msgCh})
	require.NoError(t, err)
	require.Equal(t, 1, sess.MarkedCount())

	entries := rec.Entries()
	require.True(t, hasMsg(entries, "kafka bad json"))
}

func TestConsumeClaim_EmptyOrderID_Skips(t *testing.T) {
	t.Parallel()

	rec := testlog.New()
	calls := 0

	c := &Consumer{
		logger: rec.Logger(),
		handler: func(context.Context, orders.Event) error {
			calls++
			return nil
		},
	}
	h := &groupHandler{c: c}

	dto := EventDTO{
		OrderID: "   ", 
		Status:  "created",
	}
	b, _ := json.Marshal(dto)

	sess := &fakeSession{ctx: context.Background()}
	msgCh := make(chan *sarama.ConsumerMessage, 1)
	msgCh <- &sarama.ConsumerMessage{Value: b}
	close(msgCh)

	err := h.ConsumeClaim(sess, fakeClaim{ch: msgCh})
	require.NoError(t, err)
	require.Equal(t, 1, sess.MarkedCount())
	require.Equal(t, 0, calls)

	require.True(t, hasMsg(rec.Entries(), "kafka empty order_id"))
}

func TestConsumeClaim_HandlerError_SkipsButMarks(t *testing.T) {
	t.Parallel()

	rec := testlog.New()
	sentinel := errors.New("boom")

	c := &Consumer{
		logger: rec.Logger(),
		handler: func(context.Context, orders.Event) error {
			return sentinel
		},
	}
	h := &groupHandler{c: c}

	dto := EventDTO{OrderID: "o1", Status: "created", CreatedAt: time.Now().UTC()}
	b, _ := json.Marshal(dto)

	sess := &fakeSession{ctx: context.Background()}
	msgCh := make(chan *sarama.ConsumerMessage, 1)
	msgCh <- &sarama.ConsumerMessage{Value: b}
	close(msgCh)

	err := h.ConsumeClaim(sess, fakeClaim{ch: msgCh})
	require.NoError(t, err)
	require.Equal(t, 1, sess.MarkedCount())
	require.True(t, hasMsg(rec.Entries(), "kafka handle failed, skipping message"))
}

func TestConsumeClaim_Success_Marks(t *testing.T) {
	t.Parallel()

	rec := testlog.New()
	calls := 0

	c := &Consumer{
		logger: rec.Logger(),
		handler: func(_ context.Context, ev orders.Event) error {
			calls++
			require.Equal(t, "o1", ev.OrderID)
			return nil
		},
	}
	h := &groupHandler{c: c}

	dto := EventDTO{OrderID: "o1", Status: "created"}
	b, _ := json.Marshal(dto)

	sess := &fakeSession{ctx: context.Background()}
	msgCh := make(chan *sarama.ConsumerMessage, 1)
	msgCh <- &sarama.ConsumerMessage{Value: b}
	close(msgCh)

	err := h.ConsumeClaim(sess, fakeClaim{ch: msgCh})
	require.NoError(t, err)
	require.Equal(t, 1, calls)
	require.Equal(t, 1, sess.MarkedCount())
}

func hasMsg(entries []testlog.Entry, msg string) bool {
	for _, e := range entries {
		if e.Msg == msg {
			return true
		}
	}
	return false
}
