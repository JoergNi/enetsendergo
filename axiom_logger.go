package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

const axiomRetryCapLines = 50000

type axiomEntry struct {
	t    time.Time
	line string
}

type axiomBatchWriter struct {
	dataset    string
	apiToken   string
	source     string
	baseURL    string // override for tests; empty = production Axiom
	mu         sync.Mutex
	pending    []axiomEntry
	retry      []axiomEntry
	loggedOK   bool
	httpClient *http.Client
}

func newAxiomBatchWriter(dataset, apiToken, source string) *axiomBatchWriter {
	return &axiomBatchWriter{
		dataset:    dataset,
		apiToken:   apiToken,
		source:     source,
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

// Write implements io.Writer — called synchronously from LogNormal.
// Never blocks on network I/O; just appends to the pending buffer.
func (w *axiomBatchWriter) Write(p []byte) (int, error) {
	line := strings.TrimRight(string(p), "\n")
	if line == "" {
		return len(p), nil
	}
	w.mu.Lock()
	w.pending = append(w.pending, axiomEntry{t: time.Now().UTC(), line: line})
	w.mu.Unlock()
	return len(p), nil
}

// start launches the background flush goroutine.
// On ctx cancellation a final flush drains the buffer before the goroutine exits.
func (w *axiomBatchWriter) start(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				w.flush()
				return
			case <-ticker.C:
				w.flush()
			}
		}
	}()
}

func (w *axiomBatchWriter) flush() {
	w.mu.Lock()
	if len(w.pending) == 0 && len(w.retry) == 0 {
		w.mu.Unlock()
		return
	}
	batch := append(w.retry, w.pending...)
	w.pending = nil
	w.retry = nil
	w.mu.Unlock()

	if err := w.send(batch); err != nil {
		if len(batch) > axiomRetryCapLines {
			batch = batch[len(batch)-axiomRetryCapLines:]
		}
		w.mu.Lock()
		w.retry = batch
		w.mu.Unlock()
		fmt.Fprintf(os.Stderr, "%s [%s] Axiom flush failed (%d lines buffered): %v\n",
			time.Now().Format("2006/01/02 15:04:05"), w.source, len(batch), err)
		return
	}

	w.mu.Lock()
	firstOK := !w.loggedOK
	w.loggedOK = true
	w.mu.Unlock()
	if firstOK {
		fmt.Fprintf(os.Stderr, "%s [%s] Axiom flush ok (%d lines sent)\n",
			time.Now().Format("2006/01/02 15:04:05"), w.source, len(batch))
	}
}

type axiomEvent struct {
	Time    string `json:"_time"`
	Message string `json:"message"`
	Source  string `json:"source,omitempty"`
}

func (w *axiomBatchWriter) send(batch []axiomEntry) error {
	events := make([]axiomEvent, len(batch))
	for i, e := range batch {
		events[i] = axiomEvent{
			Time:    e.t.Format(time.RFC3339Nano),
			Message: e.line,
			Source:  w.source,
		}
	}
	body, err := json.Marshal(events)
	if err != nil {
		return err
	}

	base := w.baseURL
	if base == "" {
		base = "https://api.axiom.co"
	}
	url := fmt.Sprintf("%s/v1/datasets/%s/ingest", base, w.dataset)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+w.apiToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := w.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		msg := strings.TrimSpace(string(body))
		if msg == "" {
			return fmt.Errorf("HTTP %d", resp.StatusCode)
		}
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, msg)
	}
	return nil
}
