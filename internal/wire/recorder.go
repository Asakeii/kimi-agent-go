package wire

import (
	"errors"
	"fmt"
	"io"
)

type wireRecorder struct {
	wireFile     *WireFile
	subscription *Subscription
	done         chan struct{}
	err          error
}

func newWireRecorder(wireFile *WireFile, subscription *Subscription) *wireRecorder {
	recorder := &wireRecorder{
		wireFile:     wireFile,
		subscription: subscription,
		done:         make(chan struct{}),
	}
	go recorder.consume()
	return recorder
}

func (r *wireRecorder) join() error {
	<-r.done
	return r.err
}

func (r *wireRecorder) consume() {
	defer close(r.done)
	defer r.subscription.Close()

	for {
		message, err := r.subscription.Recv()
		if errors.Is(err, io.EOF) {
			return
		}
		if err != nil {
			r.err = fmt.Errorf("wire: recorder receive: %w", err)
			return
		}
		if err := r.wireFile.AppendMessage(message); err != nil {
			r.err = fmt.Errorf("wire: recorder append: %w", err)
			return
		}
	}
}
