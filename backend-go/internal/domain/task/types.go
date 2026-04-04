package task

type TaskType string

const (
	TaskTypeAssistantAsk                TaskType = "assistant.ask"
	TaskTypeDocumentSummarize           TaskType = "document.summarize"
	TaskTypeHandoverSummarize           TaskType = "handover.summarize"
	TaskTypeDocumentExtractText         TaskType = "document.extract_text"
	TaskTypeAssistantGenerateSuggestion TaskType = "assistant.generate_suggestion"
)

type Message struct {
	RequestID   string         `json:"request_id"`
	TaskType    TaskType       `json:"task_type"`
	RelatedType string         `json:"related_type,omitempty"`
	RelatedID   string         `json:"related_id,omitempty"`
	Payload     map[string]any `json:"payload"`
}

type Result struct {
	RequestID    string         `json:"request_id"`
	Status       string         `json:"status"`
	Output       map[string]any `json:"output,omitempty"`
	ErrorMessage string         `json:"error_message,omitempty"`
}
