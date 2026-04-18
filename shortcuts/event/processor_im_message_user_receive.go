// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package event

import "context"

// ImMessageUserReceiveProcessor handles im.message.user_receive_v1 events.
type ImMessageUserReceiveProcessor struct{}

func NewImMessageUserReceiveProcessor() *ImMessageUserReceiveProcessor {
	return &ImMessageUserReceiveProcessor{}
}

func (p *ImMessageUserReceiveProcessor) EventType() string { return "im.message.user_receive_v1" }

func (p *ImMessageUserReceiveProcessor) Transform(ctx context.Context, raw *RawEvent, mode TransformMode) interface{} {
	return (&ImMessageProcessor{}).Transform(ctx, raw, mode)
}

func (p *ImMessageUserReceiveProcessor) DeduplicateKey(raw *RawEvent) string {
	return raw.Header.EventID
}
func (p *ImMessageUserReceiveProcessor) WindowStrategy() WindowConfig { return WindowConfig{} }
