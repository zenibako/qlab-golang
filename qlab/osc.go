package qlab

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/log"
	"github.com/hypebeast/go-osc/osc"
)

// Helper functions for pretty JSON logging
func logPrettyJSON(logger *log.Logger, level log.Level, message string, jsonStr string) {
	// First try to pretty print the JSON with indentation
	var jsonData any
	if err := json.Unmarshal([]byte(jsonStr), &jsonData); err != nil {
		// Fallback to raw string if JSON parsing fails
		logger.Log(level, message, "raw", jsonStr)
		return
	}

	prettyBytes, err := json.MarshalIndent(jsonData, "", "  ")
	if err != nil {
		// Fallback to structured data if pretty printing fails
		logger.Log(level, message, "data", jsonData)
		return
	}

	// Log with pretty formatted JSON
	logger.Log(level, message+"\n"+string(prettyBytes))
}

func logInfoJSON(message string, jsonStr string) {
	logPrettyJSON(log.Default(), log.InfoLevel, message, jsonStr)
}

// formatErrorWithJSON creates a pretty-printed error message from a JSON string
func formatErrorWithJSON(baseMessage string, jsonStr string) error {
	var jsonData any
	if err := json.Unmarshal([]byte(jsonStr), &jsonData); err != nil {
		// Fallback to raw string if JSON parsing fails
		return fmt.Errorf("%s: %s", baseMessage, jsonStr)
	}

	prettyBytes, err := json.MarshalIndent(jsonData, "", "  ")
	if err != nil {
		// Fallback to structured data if pretty printing fails
		return fmt.Errorf("%s: %v", baseMessage, jsonData)
	}

	return fmt.Errorf("%s:\n%s", baseMessage, string(prettyBytes))
}

type OscClient interface {
	Init(string)
	Send(string, string) []any
	GetAddress(string) string
	GetContent(string) string
}

type InitReplyArg struct {
	WorkspaceId string `json:"workspace_id"`
	Status      string `json:"status"`
	Data        string `json:"data,omitempty"`
	Error       string `json:"error,omitempty"`
}

func (q *Workspace) Send(address string, input string) []any {
	if q.dryRun && q.isWriteOperation(address) {
		log.Printf("[DRY RUN] Would send OSC message: %s ,s %s", address, input)
		return q.mockDryRunResponse(address, input)
	}
	return q.sendWithRetry(address, input, nil)
}

func (q *Workspace) SendNoReply(address string, args ...any) error {
	msg := osc.NewMessage(address)
	for _, arg := range args {
		msg.Append(arg)
	}
	log.Debugf("Sending message without reply: %s %v", address, args)
	return q.client.Send(msg)
}

func (q *Workspace) StartUpdateListener(updateHandler func(address string, args []any)) error {
	if q.updateServer != nil {
		log.Debugf("Update server already running")
		q.updateHandler = updateHandler
		return nil
	}

	q.updateHandler = updateHandler
	d := osc.NewStandardDispatcher()

	_ = d.AddMsgHandler("*", func(msg *osc.Message) {
		log.Infof("Received OSC message: %s %v", msg.Address, msg.Arguments)

		// Check if it's an update message
		if strings.HasPrefix(msg.Address, "/update") {
			log.Infof("Matched update message: %s", msg.Address)
			if q.updateHandler != nil {
				q.updateHandler(msg.Address, msg.Arguments)
			}
			return
		}

		// Check if it's a reply message
		if strings.HasPrefix(msg.Address, "/reply") {
			log.Debugf("Matched reply message: %s", msg.Address)
			// Find the first handler that matches this address (with any request ID)
			q.replyHandlersMux.Lock()
			var foundHandler chan []any
			var foundKey string
			for handlerKey, handler := range q.replyHandlers {
				// Check if this handler key matches the base address (before the #requestID)
				baseAddr := strings.Split(handlerKey, "#")[0]
				if baseAddr == msg.Address {
					log.Debugf("Routing reply to handler: %s", handlerKey)
					foundHandler = handler
					foundKey = handlerKey
					break
				}
			}
			if foundHandler != nil {
				delete(q.replyHandlers, foundKey)
			}
			q.replyHandlersMux.Unlock()

			if foundHandler != nil {
				foundHandler <- msg.Arguments
			} else {
				log.Debugf("No handler found for reply: %s", msg.Address)
			}
			return
		}
	})

	maxRetries := 10
	baseReplyPort := q.port + 1

	for i := range maxRetries {
		replyPort := baseReplyPort + i
		replyHost := fmt.Sprintf("%s:%d", q.host, replyPort)

		log.Infof("Starting persistent OSC listener on %s", replyHost)

		q.serverMux.Lock()
		q.updateServer = &osc.Server{
			Addr:       replyHost,
			Dispatcher: d,
		}
		q.updateServerReady = make(chan struct{})
		server := q.updateServer
		ready := q.updateServerReady
		q.serverMux.Unlock()

		started := make(chan error, 1)
		go func() {
			err := server.ListenAndServe()
			if err != nil && !strings.Contains(err.Error(), "use of closed network connection") {
				log.Errorf("OSC server exited with error: %v", err)
			}
			started <- err
		}()

		// Wait a bit for server to bind
		time.Sleep(50 * time.Millisecond)

		select {
		case err := <-started:
			if err != nil && strings.Contains(err.Error(), "bind: address already in use") {
				log.Debugf("Port %d in use, trying next port", replyPort)
				q.serverMux.Lock()
				close(ready) // Close channel before clearing
				q.updateServer = nil
				q.updateServerReady = nil
				q.serverMux.Unlock()
				continue
			} else if err != nil && !strings.Contains(err.Error(), "use of closed network connection") {
				log.Errorf("OSC listener error on %s: %v", replyHost, err)
				q.serverMux.Lock()
				close(ready) // Close channel before clearing
				q.updateServer = nil
				q.updateServerReady = nil
				q.serverMux.Unlock()
				continue
			}
			close(ready)
			return nil
		case <-time.After(200 * time.Millisecond):
			close(ready) // Server started successfully
			log.Infof("OSC listener started successfully on %s", replyHost)

			if err := q.SendNoReply("/updates", int32(1)); err != nil {
				log.Error("Failed to subscribe to updates", "error", err)
			} else {
				log.Info("Subscribed to QLab status updates")
			}

			return nil
		}
	}

	return fmt.Errorf("failed to start OSC listener after %d attempts", maxRetries)
}

func (q *Workspace) sendWithRetry(address string, input string, args []any) []any {
	maxRetries := q.maxRetries
	for attempt := 0; attempt <= maxRetries; attempt++ {
		msg := osc.NewMessage(address)
		if input != "" {
			msg.Append(input)
		}
		for _, arg := range args {
			msg.Append(arg)
		}

		// Generate unique request ID for this request
		q.requestCounter++
		requestID := q.requestCounter

		// Start listening for a reply with unique request ID
		reply := make(chan []any)
		q.ListenForReply(address, reply, requestID)

		// Send the message and wait for reply from listener with timeout
		startTime := time.Now()
		if err := q.client.Send(msg); err != nil {
			log.Warnf("Failed to send OSC message: %v", err)
			continue
		}
		log.Debugf("Message sent to %s:%d - %s (attempt %d/%d, requestID: %d)", q.host, q.port, msg.String(), attempt+1, maxRetries+1, requestID)

		timeout := q.timeout
		if timeout == 0 {
			timeout = 10
		}

		select {
		case result := <-reply:
			duration := time.Since(startTime)
			log.Debugf("Reply received for %s in %v (requestID: %d)", address, duration, requestID)
			q.consecutiveErrors = 0
			q.wasConnected = true
			return result
		case <-time.After(time.Duration(timeout) * time.Second):
			// Clean up the handler that timed out
			replyAddress := q.addressBuilder.BuildReplyAddress(address)
			uniqueReplyAddress := fmt.Sprintf("%s#%d", replyAddress, requestID)
			q.replyHandlersMux.Lock()
			delete(q.replyHandlers, uniqueReplyAddress)
			q.replyHandlersMux.Unlock()

			if attempt < maxRetries {
				if q.wasConnected {
					log.Warnf("Timeout waiting for reply from QLab for address %s (attempt %d/%d), retrying...", address, attempt+1, maxRetries+1)
				} else {
					log.Debugf("Timeout waiting for reply from QLab for address %s (attempt %d/%d), retrying...", address, attempt+1, maxRetries+1)
				}
				// Small delay before retry to avoid overwhelming QLab
				time.Sleep(100 * time.Millisecond)
			} else {
				q.consecutiveErrors++
				if q.wasConnected {
					log.Warnf("Timeout waiting for reply from QLab for address %s after all retry attempts", address)

					// Provide helpful guidance for common timeout scenarios
					if strings.Contains(address, "/cueLists") {
						log.Warn("The /cueLists query timed out - this usually means:")
						log.Warn("  1. Your QLab workspace has many cues (100+ cues can slow this query)")
						log.Warn("  2. QLab is busy processing other operations")
						log.Warn("  3. Network latency between client and QLab")
						log.Infof("Recommendation: Increase timeout with SetTimeout(30) or SetTimeout(60)")
						log.Infof("Current timeout: %d seconds, Current retries: %d", q.timeout, q.maxRetries)
					}

					if q.consecutiveErrors >= 2 && q.onDisconnect != nil {
						q.onDisconnect()
						q.wasConnected = false
					}
				} else {
					log.Debugf("Timeout waiting for reply from QLab for address %s after all retry attempts", address)
				}
				return []any{`{"status": "error", "error": "timeout waiting for reply from QLab"}`}
			}
		}
	}
	q.consecutiveErrors++
	if q.wasConnected && q.consecutiveErrors >= 2 && q.onDisconnect != nil {
		q.onDisconnect()
		q.wasConnected = false
	}
	return []any{`{"status": "error", "error": "timeout waiting for reply from QLab"}`}
}

func (q *Workspace) SendWithArgs(address string, args ...any) []any {
	if q.dryRun && q.isWriteOperation(address) {
		log.Printf("[DRY RUN] Would send OSC message: %s %v", address, args)
		return q.mockDryRunResponse(address, "")
	}
	return q.sendWithRetry(address, "", args)
}

func (q *Workspace) ListenForReply(address string, reply chan []any, requestID int) {
	replyAddress := q.addressBuilder.BuildReplyAddress(address)
	uniqueReplyAddress := fmt.Sprintf("%s#%d", replyAddress, requestID)

	// If persistent server is running, register handler with it
	if q.updateServer != nil {
		log.Debugf("Registering reply handler for: %s (using persistent server, requestID: %d)", replyAddress, requestID)
		q.replyHandlersMux.Lock()
		q.replyHandlers[uniqueReplyAddress] = reply
		q.replyHandlersMux.Unlock()
		return
	}

	// Note: We don't try to close existing reply servers here anymore
	// Each request gets its own server instance that closes itself after receiving a reply

	d := osc.NewStandardDispatcher()
	log.Debugf("Reply address: %s", replyAddress)

	// Capture server reference for the handler to close
	var localServer *osc.Server

	_ = d.AddMsgHandler(replyAddress, func(msg *osc.Message) {
		log.Debugf("Received reply message, closing server")
		if localServer != nil {
			_ = localServer.CloseConnection()
		}
		reply <- msg.Arguments
	})

	// Try to find an available port starting with port + 1
	maxRetries := 10
	baseReplyPort := q.port + 1

	for i := range maxRetries {
		replyPort := baseReplyPort + i
		reply_host := q.host + ":" + strconv.Itoa(replyPort)

		log.Debugf("Setting up reply server for address %s", address)
		log.Debugf("QLab host:port = %s:%d, Reply server attempting to bind to: %s", q.host, q.port, reply_host)

		server := &osc.Server{
			Addr:       reply_host,
			Dispatcher: d,
		}
		localServer = server // Assign to captured variable for handler

		// Try to start the server
		started := make(chan error, 1)
		go func() {
			started <- server.ListenAndServe()
		}()

		// Give the server a moment to start and check for port conflicts
		select {
		case err := <-started:
			if err != nil && strings.Contains(err.Error(), "bind: address already in use") {
				log.Debugf("Port %d in use, trying next port", replyPort)
				localServer = nil
				continue // Try next port
			} else if err != nil && !strings.Contains(err.Error(), "use of closed network connection") {
				log.Errorf("Reply server error on %s: %v", reply_host, err)
				localServer = nil
				continue
			}
			// If we get here, server started successfully or closed normally
			return
		case <-time.After(100 * time.Millisecond):
			// Server started without immediate error
			log.Debugf("Reply server started successfully on %s", reply_host)
			return
		}
	}

	log.Errorf("Failed to start reply server after %d attempts", maxRetries)
}
