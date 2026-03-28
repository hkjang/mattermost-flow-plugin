package main

import "sync"

type boardEventBroker struct {
	mutex              sync.RWMutex
	subscribers        map[string]map[chan BoardStreamEvent]struct{}
	summarySubscribers map[string]map[chan BoardSummaryStreamEvent]struct{}
}

func newBoardEventBroker() *boardEventBroker {
	return &boardEventBroker{
		subscribers:        map[string]map[chan BoardStreamEvent]struct{}{},
		summarySubscribers: map[string]map[chan BoardSummaryStreamEvent]struct{}{},
	}
}

func (b *boardEventBroker) Subscribe(boardID string) (<-chan BoardStreamEvent, func()) {
	ch := make(chan BoardStreamEvent, 8)

	b.mutex.Lock()
	if _, ok := b.subscribers[boardID]; !ok {
		b.subscribers[boardID] = map[chan BoardStreamEvent]struct{}{}
	}
	b.subscribers[boardID][ch] = struct{}{}
	b.mutex.Unlock()

	cancel := func() {
		b.mutex.Lock()
		defer b.mutex.Unlock()

		items := b.subscribers[boardID]
		if items == nil {
			return
		}

		if _, ok := items[ch]; ok {
			delete(items, ch)
			close(ch)
		}

		if len(items) == 0 {
			delete(b.subscribers, boardID)
		}
	}

	return ch, cancel
}

func (b *boardEventBroker) SubscribeSummary(scopeKeys []string) (<-chan BoardSummaryStreamEvent, func()) {
	keys := uniqueStrings(scopeKeys)
	ch := make(chan BoardSummaryStreamEvent, 8)

	b.mutex.Lock()
	for _, key := range keys {
		if _, ok := b.summarySubscribers[key]; !ok {
			b.summarySubscribers[key] = map[chan BoardSummaryStreamEvent]struct{}{}
		}
		b.summarySubscribers[key][ch] = struct{}{}
	}
	b.mutex.Unlock()

	cancel := func() {
		b.mutex.Lock()
		defer b.mutex.Unlock()

		closed := false
		for _, key := range keys {
			items := b.summarySubscribers[key]
			if items == nil {
				continue
			}

			if _, ok := items[ch]; ok {
				delete(items, ch)
				closed = true
			}

			if len(items) == 0 {
				delete(b.summarySubscribers, key)
			}
		}

		if closed {
			close(ch)
		}
	}

	return ch, cancel
}

func (b *boardEventBroker) Publish(event BoardStreamEvent) {
	b.mutex.RLock()
	items := b.subscribers[event.BoardID]
	channels := make([]chan BoardStreamEvent, 0, len(items))
	for ch := range items {
		channels = append(channels, ch)
	}
	b.mutex.RUnlock()

	for _, ch := range channels {
		select {
		case ch <- event:
		default:
			// Drop saturated updates instead of blocking request handling.
		}
	}
}

func (b *boardEventBroker) PublishSummary(scopeKeys []string, event BoardSummaryStreamEvent) {
	b.mutex.RLock()
	channels := make(map[chan BoardSummaryStreamEvent]struct{})
	for _, key := range uniqueStrings(scopeKeys) {
		for ch := range b.summarySubscribers[key] {
			channels[ch] = struct{}{}
		}
	}
	b.mutex.RUnlock()

	for ch := range channels {
		select {
		case ch <- event:
		default:
			// Drop saturated updates instead of blocking request handling.
		}
	}
}

func (b *boardEventBroker) Close() {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	closedSummary := map[chan BoardSummaryStreamEvent]struct{}{}
	for boardID, items := range b.subscribers {
		for ch := range items {
			close(ch)
		}
		delete(b.subscribers, boardID)
	}

	for scopeKey, items := range b.summarySubscribers {
		for ch := range items {
			if _, ok := closedSummary[ch]; ok {
				continue
			}
			close(ch)
			closedSummary[ch] = struct{}{}
		}
		delete(b.summarySubscribers, scopeKey)
	}
}

func uniqueStrings(values []string) []string {
	items := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		items = append(items, value)
	}
	return items
}
