// SPDX-License-Identifier: GPL-2.0-or-later
/*
 * Copyright (C) 2018-2022 SCANOSS.COM
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 2 of the License, or
 * (at your option) any later version.
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <https://www.gnu.org/licenses/>.
 */

package errors

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	common "github.com/scanoss/papi/api/commonv2"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// ServiceError represents a service-level error with HTTP status mapping and additional context..
type ServiceError struct {
	Message      string                 // Human-readable error message
	HTTPCode     int                    // HTTP status code to return to client
	InternalCode string                 // Internal error code for logging/monitoring
	Err          error                  // Wrapped original error for error chain
	Details      map[string]interface{} // Optional additional context
}

// Error implements the error interface.
func (e *ServiceError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

// GetHTTPCode returns the HTTP status code for this error with fallback to 500.
func (e *ServiceError) GetHTTPCode() int {
	if e.HTTPCode == 0 {
		return http.StatusInternalServerError
	}
	return e.HTTPCode
}

// Use for: missing required fields, malformed input, invalid parameters.
func NewBadRequestError(message string, err error) *ServiceError {
	return &ServiceError{
		Message:      message,
		HTTPCode:     http.StatusBadRequest,
		InternalCode: "BAD_REQUEST",
		Err:          err,
	}
}

// Use for: ecosystem not found, dependencies not found, resource missing.
func NewNotFoundError(resource string) *ServiceError {
	return &ServiceError{
		Message:      fmt.Sprintf("%s not found", resource),
		HTTPCode:     http.StatusNotFound,
		InternalCode: "NOT_FOUND",
		Err:          nil,
	}
}

// Use for: unexpected errors, programming errors, unhandled exceptions.
func NewInternalError(message string, err error) *ServiceError {
	return &ServiceError{
		Message:      message,
		HTTPCode:     http.StatusInternalServerError,
		InternalCode: "INTERNAL_ERROR",
		Err:          err,
	}
}

// Use for: database down, external service timeout, rate limits exceeded.
func NewServiceUnavailableError(message string, err error) *ServiceError {
	return &ServiceError{
		Message:      message,
		HTTPCode:     http.StatusServiceUnavailable,
		InternalCode: "SERVICE_UNAVAILABLE",
		Err:          err,
	}
}

// IsServiceError checks if an error is a ServiceError.
func IsServiceError(err error) bool {
	var serviceErr *ServiceError
	return errors.As(err, &serviceErr)
}

// GetServiceError extracts a ServiceError from an error chain.
func GetServiceError(err error) (*ServiceError, bool) {
	var serviceErr *ServiceError
	if errors.As(err, &serviceErr) {
		return serviceErr, true
	}
	return nil, false
}

// HandleServiceError converts a ServiceError to a gRPC response with proper HTTP status.
func HandleServiceError(ctx context.Context, s *zap.SugaredLogger, err error) *common.StatusResponse {
	var serviceErr *ServiceError
	if IsServiceError(err) {
		serviceErr, _ = GetServiceError(err)
		// Set HTTP trailer based on custom error
		trailerErr := grpc.SetTrailer(ctx, metadata.Pairs("x-http-code", fmt.Sprintf("%d", serviceErr.GetHTTPCode())))
		if trailerErr != nil {
			s.Debugf("error setting x-http-code to trailer: %v", trailerErr)
		}

		// Log with structured data for monitoring
		s.Errorw("service error",
			"error", serviceErr.Error(),
			"http_code", serviceErr.GetHTTPCode(),
			"internal_code", serviceErr.InternalCode,
			"details", serviceErr.Details,
		)

		return &common.StatusResponse{
			Status:  common.StatusCode_FAILED,
			Message: serviceErr.Message,
		}
	}

	// Default to 500 for unknown errors
	trailerErr := grpc.SetTrailer(ctx, metadata.Pairs("x-http-code", "500"))
	if trailerErr != nil {
		s.Debugf("error setting x-http-code to trailer: %v", trailerErr)
	}

	s.Errorw("unhandled error", "error", err.Error())

	return &common.StatusResponse{
		Status:  common.StatusCode_FAILED,
		Message: "internal server error",
	}
}
