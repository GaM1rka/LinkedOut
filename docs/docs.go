package docs

import "github.com/swaggo/swag"

const docTemplate = `{
  "swagger": "2.0",
  "info": {
    "description": "REST API for LinkedOut Telegram bot MVP metrics.",
    "title": "LinkedOut Metrics API",
    "version": "1.0"
  },
  "basePath": "/",
  "paths": {
    "/health": {
      "get": {
        "tags": ["health"],
        "summary": "Healthcheck",
        "responses": {
          "200": {"description": "OK", "schema": {"type": "object"}},
          "503": {"description": "Service Unavailable", "schema": {"$ref": "#/definitions/errorResponse"}}
        }
      }
    },
    "/api/v1/users/upsert": {
      "post": {
        "tags": ["users"],
        "summary": "Upsert Telegram user",
        "consumes": ["application/json"],
        "produces": ["application/json"],
        "parameters": [{"in": "body", "name": "request", "required": true, "schema": {"$ref": "#/definitions/UpsertUserRequest"}}],
        "responses": {"200": {"description": "OK", "schema": {"$ref": "#/definitions/User"}}}
      }
    },
    "/api/v1/interviews/start": {
      "post": {
        "tags": ["interviews"],
        "summary": "Start interview",
        "consumes": ["application/json"],
        "produces": ["application/json"],
        "parameters": [{"in": "body", "name": "request", "required": true, "schema": {"$ref": "#/definitions/StartInterviewRequest"}}],
        "responses": {"200": {"description": "OK", "schema": {"$ref": "#/definitions/InterviewStarted"}}}
      }
    },
    "/api/v1/interviews/{interview_id}/answers": {
      "post": {
        "tags": ["interviews"],
        "summary": "Save interview answer",
        "consumes": ["application/json"],
        "produces": ["application/json"],
        "parameters": [
          {"in": "path", "name": "interview_id", "required": true, "type": "string"},
          {"in": "body", "name": "request", "required": true, "schema": {"$ref": "#/definitions/AnswerRequest"}}
        ],
        "responses": {"200": {"description": "OK", "schema": {"type": "object"}}}
      }
    },
    "/api/v1/interviews/{interview_id}/complete": {
      "post": {
        "tags": ["interviews"],
        "summary": "Complete interview",
        "consumes": ["application/json"],
        "produces": ["application/json"],
        "parameters": [
          {"in": "path", "name": "interview_id", "required": true, "type": "string"},
          {"in": "body", "name": "request", "required": false, "schema": {"$ref": "#/definitions/CompleteInterviewRequest"}}
        ],
        "responses": {"200": {"description": "OK", "schema": {"$ref": "#/definitions/InterviewCompleted"}}}
      }
    },
    "/api/v1/generations": {
      "post": {
        "tags": ["generations"],
        "summary": "Save generated resume description",
        "consumes": ["application/json"],
        "produces": ["application/json"],
        "parameters": [{"in": "body", "name": "request", "required": true, "schema": {"$ref": "#/definitions/GenerationRequest"}}],
        "responses": {"200": {"description": "OK", "schema": {"$ref": "#/definitions/GenerationCreated"}}}
      }
    },
    "/api/v1/feedbacks": {
      "post": {
        "tags": ["feedbacks"],
        "summary": "Save user feedback",
        "consumes": ["application/json"],
        "produces": ["application/json"],
        "parameters": [{"in": "body", "name": "request", "required": true, "schema": {"$ref": "#/definitions/FeedbackRequest"}}],
        "responses": {"200": {"description": "OK", "schema": {"type": "object"}}}
      }
    },
    "/api/v1/manual-reviews": {
      "post": {
        "tags": ["manual_reviews"],
        "summary": "Save manual review",
        "consumes": ["application/json"],
        "produces": ["application/json"],
        "parameters": [{"in": "body", "name": "request", "required": true, "schema": {"$ref": "#/definitions/ManualReviewRequest"}}],
        "responses": {"200": {"description": "OK", "schema": {"type": "object"}}}
      }
    },
    "/api/v1/manual-reviews/pending": {
      "get": {
        "tags": ["manual_reviews"],
        "summary": "List pending manual reviews",
        "produces": ["application/json"],
        "parameters": [{"in": "query", "name": "limit", "required": false, "type": "integer"}],
        "responses": {"200": {"description": "OK", "schema": {"type": "array", "items": {"$ref": "#/definitions/PendingReview"}}}}
      }
    },
    "/api/v1/metrics/summary": {
      "get": {
        "tags": ["metrics"],
        "summary": "MVP metrics summary",
        "produces": ["application/json"],
        "responses": {"200": {"description": "OK", "schema": {"$ref": "#/definitions/MetricsSummary"}}}
      }
    },
    "/api/v1/events": {
      "post": {
        "tags": ["events"],
        "summary": "Save product event",
        "consumes": ["application/json"],
        "produces": ["application/json"],
        "parameters": [{"in": "body", "name": "request", "required": true, "schema": {"$ref": "#/definitions/EventRequest"}}],
        "responses": {"200": {"description": "OK", "schema": {"type": "object"}}}
      }
    }
  },
  "definitions": {
    "errorResponse": {
      "type": "object",
      "properties": {"error": {"type": "string"}}
    },
    "User": {
      "type": "object",
      "properties": {
        "id": {"type": "string"},
        "telegram_id": {"type": "integer"},
        "username": {"type": "string"},
        "first_name": {"type": "string"},
        "created_at": {"type": "string", "format": "date-time"}
      }
    },
    "UpsertUserRequest": {
      "type": "object",
      "required": ["telegram_id"],
      "properties": {
        "telegram_id": {"type": "integer", "example": 123},
        "username": {"type": "string", "example": "user"},
        "first_name": {"type": "string", "example": "Amir"}
      }
    },
    "StartInterviewRequest": {
      "type": "object",
      "required": ["telegram_id", "target_role", "project_type"],
      "properties": {
        "telegram_id": {"type": "integer", "example": 123},
        "target_role": {"type": "string", "example": "Backend"},
        "project_type": {"type": "string", "example": "Pet-project"}
      }
    },
    "InterviewStarted": {
      "type": "object",
      "properties": {
        "interview_id": {"type": "string"},
        "status": {"type": "string"},
        "started_at": {"type": "string", "format": "date-time"}
      }
    },
    "AnswerRequest": {
      "type": "object",
      "required": ["question_order", "question_text", "answer_text"],
      "properties": {
        "question_order": {"type": "integer", "example": 1},
        "question_text": {"type": "string"},
        "answer_text": {"type": "string"}
      }
    },
    "CompleteInterviewRequest": {
      "type": "object",
      "properties": {"completed_at": {"type": "string", "format": "date-time"}}
    },
    "InterviewCompleted": {
      "type": "object",
      "properties": {
        "interview_id": {"type": "string"},
        "duration_seconds": {"type": "integer", "example": 820}
      }
    },
    "GenerationRequest": {
      "type": "object",
      "required": ["interview_id", "bullets", "project_summary", "skills", "risk_warnings"],
      "properties": {
        "interview_id": {"type": "string"},
        "bullets": {"type": "string"},
        "project_summary": {"type": "string"},
        "skills": {"type": "string"},
        "risk_warnings": {"type": "string"},
        "raw_llm_response": {"type": "string"}
      }
    },
    "GenerationCreated": {
      "type": "object",
      "properties": {"generation_id": {"type": "string"}}
    },
    "FeedbackRequest": {
      "type": "object",
      "required": ["interview_id", "generation_id", "ready_to_use", "needs_major_edits", "payment_willingness"],
      "properties": {
        "interview_id": {"type": "string"},
        "generation_id": {"type": "string"},
        "ready_to_use": {"type": "boolean"},
        "needs_major_edits": {"type": "boolean"},
        "issue_type": {"type": "string", "enum": ["none", "water", "inaccuracy", "exaggeration", "other"]},
        "payment_willingness": {"type": "string", "enum": ["none", "maybe", "one_project", "full_resume"]},
        "comment": {"type": "string"}
      }
    },
    "ManualReviewRequest": {
      "type": "object",
      "required": ["generation_id", "reviewer_telegram_id", "status"],
      "properties": {
        "generation_id": {"type": "string"},
        "reviewer_telegram_id": {"type": "integer"},
        "status": {"type": "string", "enum": ["pass", "fail_water", "fail_inaccuracy", "fail_exaggeration", "fail_other"]},
        "comment": {"type": "string"}
      }
    },
    "PendingReview": {
      "type": "object",
      "properties": {
        "generation_id": {"type": "string"},
        "telegram_id": {"type": "integer"},
        "target_role": {"type": "string"},
        "project_type": {"type": "string"},
        "bullets": {"type": "string"},
        "project_summary": {"type": "string"},
        "created_at": {"type": "string", "format": "date-time"}
      }
    },
    "MetricsSummary": {
      "type": "object",
      "properties": {
        "started_interviews": {"type": "integer"},
        "completed_interviews": {"type": "integer"},
        "completion_rate": {"type": "number"},
        "avg_interview_duration_seconds": {"type": "number"},
        "avg_interview_duration_minutes": {"type": "number"},
        "ready_to_use_rate": {"type": "number"},
        "ready_to_use_count": {"type": "integer"},
        "payment_willing_users": {"type": "integer"},
        "manual_review_total": {"type": "integer"},
        "manual_review_passed": {"type": "integer"},
        "manual_review_pass_rate": {"type": "number"},
        "mvp_success": {"type": "boolean"},
        "not_enough_data": {"type": "boolean"},
        "targets": {"type": "object"}
      }
    },
    "EventRequest": {
      "type": "object",
      "required": ["event_type"],
      "properties": {
        "telegram_id": {"type": "integer"},
        "interview_id": {"type": "string"},
        "event_type": {"type": "string"},
        "payload": {"type": "object"}
      }
    }
  }
}`

var SwaggerInfo = &swag.Spec{
	Version:          "1.0",
	Host:             "",
	BasePath:         "/",
	Schemes:          []string{},
	Title:            "LinkedOut Metrics API",
	Description:      "REST API for LinkedOut Telegram bot MVP metrics.",
	InfoInstanceName: "swagger",
	SwaggerTemplate:  docTemplate,
	LeftDelim:        "{{",
	RightDelim:       "}}",
}

func init() {
	swag.Register(SwaggerInfo.InstanceName(), SwaggerInfo)
}
