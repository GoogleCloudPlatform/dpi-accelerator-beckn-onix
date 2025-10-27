// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"

	"github.com/google/dpi-accelerator-beckn-onix/pkg/model"
)

var ErrLROAlreadyProcessed = errors.New("LRO_ALREADY_PROCESSED")

// encrypter defines the methods for encryption.
type encrypterSrv interface {
	Encrypt(ctx context.Context, data string, npKey string) (string, error)
}

// npClient defines the interface for communicating with a Network Participant.
type npClient interface {
	OnSubscribe(ctx context.Context, callbackURL string, request *model.OnSubscribeRequest) (*model.OnSubscribeResponse, error)
}

// challengeSrv handles generation and verification of challenges.
type challengeSrv interface {
	NewChallenge() (string, error)
	Verify(challenge, answer string) bool
}

type regRepo interface {
	GetOperation(context.Context, string) (*model.LRO, error)
	UpdateOperation(context.Context, *model.LRO) (*model.LRO, error)
	UpsertSubscriptionAndLRO(ctx context.Context, sub *model.Subscription, lro *model.LRO) (*model.Subscription, *model.LRO, error)
	Lookup(ctx context.Context, sub *model.Subscription) ([]model.Subscription, error)
}

type adminEventPublisher interface {
	PublishSubscriptionRequestApprovedEvent(ctx context.Context, req *model.LRO) (string, error)
	PublishSubscriptionRequestRejectedEvent(ctx context.Context, req *model.LRO) (string, error)
}

type adminService struct {
	cfg         *AdminConfig
	regRepo     regRepo
	chSrv       challengeSrv
	encryptor   encrypterSrv
	npClient    npClient
	evPublisher adminEventPublisher
}

type AdminConfig struct {
	OperationRetryMax int `yaml:"operationRetryMax"`
}

// NewAdminService creates a new adminService.
func NewAdminService(regRepo regRepo, chSrv challengeSrv, encryptor encrypterSrv, npClient npClient, evPub adminEventPublisher, cfg *AdminConfig) (*adminService, error) {
	if regRepo == nil {
		slog.Error("NewAdminService: regRepo cannot be nil")
		return nil, errors.New("regRepo cannot be nil")
	}
	if chSrv == nil {
		slog.Error("NewAdminService: challengeService cannot be nil")
		return nil, errors.New("challengeService cannot be nil")
	}
	if encryptor == nil {
		slog.Error("NewAdminService: encryptor cannot be nil")
		return nil, errors.New("encryptor cannot be nil")
	}

	if npClient == nil {
		slog.Error("NewAdminService: npClient cannot be nil")
		return nil, errors.New("npClient cannot be nil")
	}
	if cfg == nil {
		slog.Error("NewAdminService: AdminConfig cannot be nil")
		return nil, errors.New("AdminConfig cannot be nil")
	}
	if cfg.OperationRetryMax <= 0 {
		slog.Error("NewAdminService: OperationRetryMax cannot be zero or negative")
		return nil, errors.New("AdminConfig.OperationRetryMax cannot be zero or negative")
	}

	if evPub == nil {
		slog.Error("NewAdminService: eventPublisher cannot be nil")
		return nil, errors.New("eventPublisher cannot be nil")
	}
	return &adminService{regRepo: regRepo, chSrv: chSrv, encryptor: encryptor, npClient: npClient, evPublisher: evPub, cfg: cfg}, nil
}

// ApproveSubscription approves a pending subscription LRO.
func (s *adminService) ApproveSubscription(ctx context.Context, req *model.OperationActionRequest) (*model.Subscription, *model.LRO, error) {
	if req == nil {
		slog.ErrorContext(ctx, "AdminService: OperationActionRequest cannot be nil")
		return nil, nil, errors.New("OperationActionRequest cannot be nil")
	}
	if req.OperationID == "" {
		slog.ErrorContext(ctx, "AdminService: OperationID cannot be empty")
		return nil, nil, errors.New("OperationID cannot be empty")

	}
	slog.InfoContext(ctx, "AdminService: Starting subscription approval process", "operation_id", req.OperationID)

	lro, err := s.lro(ctx, req.OperationID)
	if err != nil {
		return nil, nil, err
	}
	subReq, err := s.subReq(ctx, lro)
	if err != nil {
		return nil, nil, err
	}

	sub := &model.Subscription{
		Subscriber: model.Subscriber{
			SubscriberID: subReq.SubscriberID,
			Domain:       subReq.Domain,
			Type:         subReq.Type,
		},
	}
	subs, err := s.regRepo.Lookup(ctx, sub)
	if err != nil {
		slog.ErrorContext(ctx, "AdminService: lookup failed", "error", err)
		return nil, nil, fmt.Errorf("lookup failed: %w", err)
	}
	slog.Debug("AdminService: lookup successful", "len", len(subs), "lro_type", lro.Type)
	if len(subs) > 0 && lro.Type == model.OperationTypeCreateSubscription {
		err := fmt.Errorf("subscription already exists: subscriber_id '%s', domain '%s', type '%s'", subReq.SubscriberID, subReq.Domain, subReq.Type)
		slog.ErrorContext(ctx, "AdminService: Subscription already exists", "subscriber_id", subReq.SubscriberID, "domain", subReq.Domain, "type", subReq.Type)
		if updateErr := s.updateLROError(ctx, lro, err, model.LROStatusFailure); updateErr != nil {
			slog.ErrorContext(ctx, "AdminService: Failed to update LRO with failure status", "operation_id", lro.OperationID, "update_error", updateErr)
		}
		return nil, nil, err
	}
	if len(subs) == 0 && lro.Type == model.OperationTypeUpdateSubscription {
		err := fmt.Errorf("subscription does not exists: subscriber_id '%s', domain '%s', type '%s'", subReq.SubscriberID, subReq.Domain, subReq.Type)
		slog.ErrorContext(ctx, "AdminService: Subscription does not exists", "subscriber_id", subReq.SubscriberID, "domain", subReq.Domain, "type", subReq.Type)
		if updateErr := s.updateLROError(ctx, lro, err, model.LROStatusFailure); updateErr != nil {
			slog.ErrorContext(ctx, "AdminService: Failed to update LRO with failure status", "operation_id", lro.OperationID, "update_error", updateErr)
		}
		return nil, nil, err
	}

	challenge, encryptedChallenge, err := s.challenge(ctx, lro, subReq.EncrPublicKey)
	if err != nil {
		// generateAndEncryptChallenge logs and updates LRO
		return nil, nil, err
	}

	onSubscribeResp, err := s.onSubscribe(ctx, lro, subReq, encryptedChallenge)
	if err != nil {
		return nil, nil, err
	}

	if err := s.verifyChallenge(ctx, lro, challenge, onSubscribeResp.Answer); err != nil {
		// verifyChallengeResponse logs and updates LRO
		return nil, nil, err
	}

	return s.approve(ctx, lro, subReq)
}

// lro retrieves the LRO and performs initial validations.
func (s *adminService) lro(ctx context.Context, operationID string) (*model.LRO, error) {
	lro, err := s.regRepo.GetOperation(ctx, operationID)
	if err != nil || lro == nil {
		slog.ErrorContext(ctx, "AdminService: Failed to get LRO ", "operation_id", operationID, "error", err)
		return nil, fmt.Errorf("failed to get LRO: %w", err)
	}

	if lro.RetryCount > s.cfg.OperationRetryMax {
		slog.ErrorContext(ctx, "AdminService: Max retries exceeded for operation", "operation_id", operationID, "retry_count", lro.RetryCount)
		return lro, errors.New("max retries exceeded for operation")
	}

	if lro.Type != model.OperationTypeCreateSubscription && lro.Type != model.OperationTypeUpdateSubscription {
		slog.WarnContext(ctx, "AdminService: Attempted to process non-subscription LRO", "operation_id", operationID, "type", lro.Type)
		return lro, fmt.Errorf("invalid operation type: %s, expected CREATE_SUBSCRIPTION or UPDATE_SUBSCRIPTION", lro.Type)
	}

	if lro.Status == model.LROStatusApproved || lro.Status == model.LROStatusRejected {
		slog.WarnContext(ctx, "AdminService: LRO has already been processed", "operation_id", operationID, "status", lro.Status)
		return lro, fmt.Errorf("%w: operation %s has status %s", ErrLROAlreadyProcessed, operationID, lro.Status)
	}
	return lro, nil
}

// subReq unmarshals and validates the SubscriptionRequest from LRO.
func (s *adminService) subReq(ctx context.Context, lro *model.LRO) (*model.SubscriptionRequest, error) {
	var subReq model.SubscriptionRequest
	if err := json.Unmarshal(lro.RequestJSON, &subReq); err != nil {
		slog.ErrorContext(ctx, "AdminService: Failed to unmarshal LRO request JSON", "operation_id", lro.OperationID, "error", err)
		err := fmt.Errorf("failed to unmarshal LRO request JSON: %w", err)
		if updateErr := s.updateLROError(ctx, lro, err, model.LROStatusRejected); updateErr != nil {
			slog.ErrorContext(ctx, "AdminService: Failed to update LRO with failure status", "operation_id", lro.OperationID, "update_error", updateErr)
		}
		return nil, err
	}

	if subReq.URL == "" {
		slog.ErrorContext(ctx, "AdminService: Callback URL missing in subscription request", "operation_id", lro.OperationID)
		err := errors.New("callback URL missing in subscription request")
		if updateErr := s.updateLROError(ctx, lro, err, model.LROStatusRejected); updateErr != nil {
			slog.ErrorContext(ctx, "AdminService: Failed to update LRO with failure status", "operation_id", lro.OperationID, "update_error", updateErr)
		}
		return nil, err
	}
	if subReq.EncrPublicKey == "" {
		slog.ErrorContext(ctx, "AdminService: Encryption public key missing in subscription request", "operation_id", lro.OperationID)
		err := errors.New("encryption public key missing")
		if updateErr := s.updateLROError(ctx, lro, err, model.LROStatusRejected); updateErr != nil {
			slog.ErrorContext(ctx, "AdminService: Failed to update LRO with failure status", "operation_id", lro.OperationID, "update_error", updateErr)
		}
		return nil, err
	}
	return &subReq, nil
}

// challenge handles challenge generation and encryption.
func (s *adminService) challenge(ctx context.Context, lro *model.LRO, subscriberEncrPublicKey string) (string, string, error) {
	challenge, err := s.chSrv.NewChallenge()
	if err != nil {
		slog.ErrorContext(ctx, "AdminService: Failed to generate challenge", "operation_id", lro.OperationID, "error", err)
		err := fmt.Errorf("failed to generate challenge: %w", err)
		if updateErr := s.updateLROError(ctx, lro, err, model.LROStatusFailure); updateErr != nil {
			slog.ErrorContext(ctx, "AdminService: Failed to update LRO with failure status", "operation_id", lro.OperationID, "update_error", updateErr)
		}
		return "", "", err
	}

	encryptedChallenge, err := s.encryptor.Encrypt(ctx, challenge, subscriberEncrPublicKey)
	if err != nil {
		slog.ErrorContext(ctx, "AdminService: Failed to encrypt challenge", "operation_id", lro.OperationID, "error", err)
		err := fmt.Errorf("failed to encrypt challenge: %w", err)
		if updateErr := s.updateLROError(ctx, lro, err, model.LROStatusFailure); updateErr != nil {
			slog.ErrorContext(ctx, "AdminService: Failed to update LRO with failure status", "operation_id", lro.OperationID, "update_error", updateErr)
		}
		return "", "", err
	}
	return challenge, encryptedChallenge, nil
}

// onSubscribe makes the HTTP call to the Network Participant.
func (s *adminService) onSubscribe(ctx context.Context, lro *model.LRO, subReq *model.SubscriptionRequest, encryptedChallenge string) (*model.OnSubscribeResponse, error) {
	onSubscribeReq := &model.OnSubscribeRequest{Challenge: encryptedChallenge, MessageID: subReq.MessageID}
	onSubscribeResp, err := s.npClient.OnSubscribe(ctx, subReq.URL, onSubscribeReq)
	if err != nil {
		slog.WarnContext(ctx, "AdminService: /on_subscribe callback failed", "operation_id", lro.OperationID, "callback_url", subReq.URL, "error", err)
		err := fmt.Errorf("network Participant /on_subscribe callback failed: %w", err)
		if updateErr := s.updateLROError(ctx, lro, err, model.LROStatusFailure); updateErr != nil {
			slog.ErrorContext(ctx, "AdminService: Failed to update LRO with failure status", "operation_id", lro.OperationID, "update_error", updateErr)
		}
		return nil, err
	}
	return onSubscribeResp, nil
}

// verifyChallenge verifies the NP's answer to the challenge.
func (s *adminService) verifyChallenge(ctx context.Context, lro *model.LRO, challenge, answer string) error {
	if !s.chSrv.Verify(challenge, answer) {
		slog.WarnContext(ctx, "AdminService: Challenge mismatch from /on_subscribe response", "operation_id", lro.OperationID)
		err := errors.New("challenge verification failed")
		if updateErr := s.updateLROError(ctx, lro, err, model.LROStatusFailure); updateErr != nil {
			slog.ErrorContext(ctx, "AdminService: Failed to update LRO with failure status", "operation_id", lro.OperationID, "update_error", updateErr)
		}
		return err
	}
	slog.InfoContext(ctx, "AdminService: Challenge verification successful", "operation_id", lro.OperationID)
	return nil
}

// approve updates subscription and LRO status to approved/succeeded.
func (s *adminService) approve(ctx context.Context, lro *model.LRO, subReq *model.SubscriptionRequest) (*model.Subscription, *model.LRO, error) {
	subReq.Status = model.SubscriptionStatusSubscribed
	lro.Status = model.LROStatusApproved
	sub, updatedLRO, err := s.regRepo.UpsertSubscriptionAndLRO(ctx, &subReq.Subscription, lro)
	if err != nil {
		slog.ErrorContext(ctx, "AdminService: Failed to upsert subscription and update LRO", "operation_id", lro.OperationID, "error", err)
		return nil, lro, err
	}
	slog.InfoContext(ctx, "AdminService: Subscription approved and LRO updated successfully", "operation_id", updatedLRO.OperationID)
	evID, err := s.evPublisher.PublishSubscriptionRequestApprovedEvent(ctx, updatedLRO)
	if err != nil {
		slog.ErrorContext(ctx, "AdminService: Failed to publish subscription approved event", "error", err)
	} else {
		slog.InfoContext(ctx, "AdminService: Published subscription approved event", "operation_id", updatedLRO.OperationID, "event_id", evID)
	}
	return sub, updatedLRO, nil
}
func (s *adminService) updateLROError(ctx context.Context, lro *model.LRO, originalErr error, status model.LROStatus) error {
	errorPayload := map[string]string{"error": originalErr.Error()}
	errJson, marshalErr := json.Marshal(errorPayload)
	if marshalErr != nil {
		slog.ErrorContext(ctx, "AdminService:updateLROError - failed to marshal error", "error", marshalErr)
		return fmt.Errorf("AdminService:updateLROError - failed to marshal error : %w", marshalErr)
	}
	lro.ErrorDataJSON = errJson
	lro.RetryCount++
	lro.Status = status
	if lro.RetryCount > s.cfg.OperationRetryMax {
		lro.Status = model.LROStatusRejected
	}
	_, updateErr := s.regRepo.UpdateOperation(ctx, lro)
	if updateErr != nil {
		slog.ErrorContext(ctx, "AdminService: CRITICAL ERROR - Failed to update LRO with error status after original failure", "operation_id", lro.OperationID, "original_error", originalErr, "update_error", updateErr)
		// If this fails, we're in a bad state, but we should still return the original processing error.
		return fmt.Errorf("failed to update LRO status after processing error: %w (original error: %v)", updateErr, originalErr)
	}
	return nil
}

// RejectSubscription rejects a pending subscription LRO.
func (s *adminService) RejectSubscription(ctx context.Context, req *model.OperationActionRequest) (*model.LRO, error) {

	if req == nil {
		slog.ErrorContext(ctx, "AdminService: OperationActionRequest cannot be nil")
		return nil, errors.New("OperationActionRequest cannot be nil")
	}
	if req.OperationID == "" {
		slog.ErrorContext(ctx, "AdminService: OperationID cannot be empty")
		return nil, errors.New("OperationID cannot be empty")
	}
	if req.Reason == "" {
		slog.ErrorContext(ctx, "AdminService: Reason cannot be empty")
		return nil, errors.New("reason cannot be empty")
	}
	operationID := req.OperationID
	reason := req.Reason

	slog.InfoContext(ctx, "LROService: Rejecting subscription", "operation_id", operationID, "reason", reason)

	lro, err := s.lro(ctx, operationID)
	if err != nil {
		return nil, err
	}
	lro.Status = model.LROStatusRejected
	errorPayload := map[string]string{"reason": reason}
	resJson, err := json.Marshal(errorPayload)
	if err != nil {
		slog.ErrorContext(ctx, "AdminService:RejectSubscription - failed to marshal reason json", "error", err)
		return nil, fmt.Errorf("AdminService:RejectSubscription - failed to marshal reason json: %w", err)
	}
	lro.ErrorDataJSON = resJson

	updatedLRO, err := s.regRepo.UpdateOperation(ctx, lro)
	if err != nil {
		slog.ErrorContext(ctx, "AdminService:RejectSubscription - Failed to update LRO", "operation_id", lro.OperationID, "error", err)
		return nil, fmt.Errorf("AdminService:RejectSubscription - failed to update LRO error: %w", err)
	}
	if evID, err := s.evPublisher.PublishSubscriptionRequestRejectedEvent(ctx, updatedLRO); err != nil {
		slog.ErrorContext(ctx, "AdminService: Failed to publish subscription rejected event", "error", err)
	} else {
		slog.InfoContext(ctx, "AdminService: Published subscription rejected event", "operation_id", updatedLRO.OperationID, "event_id", evID)
	}
	return updatedLRO, nil
}
