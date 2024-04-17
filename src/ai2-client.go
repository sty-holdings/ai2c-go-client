// Package src
/*
This is a client for STY Holdings services

RESTRICTIONS:
	None

NOTES:
    None

COPYRIGHT:
	Copyright 2022
	Licensed under the Apache License, Version 2.0 (the "License");
	you may not use this file except in compliance with the License.
	You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

	Unless required by applicable law or agreed to in writing, software
	distributed under the License is distributed on an "AS IS" BASIS,
	WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
	See the License for the specific language governing permissions and
	limitations under the License.

*/
package src

import (
	"encoding/json"
	"fmt"
	"runtime"
	"strconv"
	"strings"
	"time"

	awsSSM "github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/nats-io/nats.go"
	ctv "github.com/sty-holdings/constant-type-vars-go/v2024"
	awss "github.com/sty-holdings/sty-shared/v2024/awsServices"
	cfgs "github.com/sty-holdings/sty-shared/v2024/configuration"
	hv "github.com/sty-holdings/sty-shared/v2024/helpersValidators"
	jwts "github.com/sty-holdings/sty-shared/v2024/jwtServices"
	ns "github.com/sty-holdings/sty-shared/v2024/natsSerices"
	pi "github.com/sty-holdings/sty-shared/v2024/programInfo"
)

//goland:noinspection ALL
const (
	PROGRAM_NAME              = "ai2c-go-client"
	AI2C_SSM_PARAMETER_PREFIX = "ai2c"
)

type Ai2CClient struct {
	awsSettings        awss.AWSSettings
	environment        string
	natsService        ns.NATSService
	natsConfig         ns.NATSConfiguration
	secretKey          string
	styhCustomerConfig styhCustomerConfig
	tempDirectory      string
}

type Ai2CPaymentInfo struct {
	Amount                    float64  `json:"amount,omitempty"`
	UseAutomaticPaymentMethod bool     `json:"use_automatic_payment_method,omitempty"`
	CancellationReason        string   `json:"cancellation_reason,omitempty"`
	CaptureFunds              string   `json:"capture_funds,omitempty"`
	Currency                  string   `json:"currency,omitempty"`
	CustomerId                string   `json:"customer_id,omitempty"`
	Description               string   `json:"description,omitempty"`
	ReturnRecordsLimit        int64    `json:"return_records_limit,omitempty"`
	PaymentIntentId           string   `json:"id,omitempty"`
	PaymentMethod             string   `json:"payment_method,omitempty"`
	ReceiptEmail              string   `json:"receipt_email,omitempty"`
	ReturnURL                 string   `json:"return_url,omitempty,omitempty"`
	StartingAfterRecord       string   `json:"starting_after_record,omitempty"`
	Keys                      SaaSKeys `json:"keys,omitempty"`
}

type CancelPaymentIntentRequest struct {
	SaaSKey            string `json:"saas_key"`
	PaymentIntentId    string `json:"id"`
	CancellationReason string `json:"cancellation_reason"`
}

type ListPaymentIntentRequest struct {
	SaaSKey       string `json:"saas_key"`
	CustomerId    string `json:"customer_id,omitempty"`
	Limit         int64  `json:"limit,omitempty"`
	StartingAfter string `json:"starting_after,omitempty"`
}

type ListPaymentMethodRequest struct {
	SaaSKey string `json:"saas_key"`
}

type PaymentIntentRequest struct {
	Amount                  float64 `json:"amount"`
	AutomaticPaymentMethods bool    `json:"automatic_payment_methods,omitempty"`
	Currency                string  `json:"currency"`
	Description             string  `json:"description,omitempty"`
	SaaSKey                 string  `json:"saas_key"`
	ReceiptEmail            string  `json:"receipt_email"`
	ReturnURL               string  `json:"return_url,omitempty"`
	// Confirm            bool     `json:"confirm,omitempty"`
	// PaymentMethodTypes []string `json:"payment_method_types,omitempty"`
}

type SaaSKeys struct {
	Public string `json:"public_key,omitempty"`
	Secret string `json:"secret_key,omitempty"`
}

type styhCustomerConfig struct {
	clientId  string
	secretKey string
	tokens    awss.CognitoTokens
	username  string
}

func NewAI2CClient(styhClientId, environment, password, secretKey, tempDirectory, username, configFileFQN string) (
	ai2cClientPtr Ai2CClient,
	errorInfo pi.ErrorInfo,
) {

	var (
		tEnvironment   string
		tPassword      string
		tSecretKey     string
		tSTYHClientId  string
		tTempDirectory string
		tUsername      string
	)

	var (
		tConfigMap = make(map[string]interface{})
	)

	if configFileFQN == ctv.VAL_EMPTY {
		if styhClientId == ctv.VAL_EMPTY {
			errorInfo = pi.NewErrorInfo(pi.ErrRequiredArgumentMissing, fmt.Sprintf("%v%v", ctv.TXT_MISSING_PARAMETER, ctv.FN_CLIENT_ID))
			return
		} else {
			tSTYHClientId = styhClientId
		}
		if password == ctv.VAL_EMPTY {
			errorInfo = pi.NewErrorInfo(pi.ErrRequiredArgumentMissing, fmt.Sprintf("%v%v", ctv.TXT_MISSING_PARAMETER, ctv.FN_PASSWORD))
			return
		} else {
			tPassword = password
		}
		if secretKey == ctv.VAL_EMPTY {
			errorInfo = pi.NewErrorInfo(pi.ErrRequiredArgumentMissing, fmt.Sprintf("%v%v", ctv.TXT_MISSING_PARAMETER, ctv.FN_SECRET_KEY))
			return
		} else {
			tSecretKey = secretKey
		}
		if tempDirectory == ctv.VAL_EMPTY {
			errorInfo = pi.NewErrorInfo(pi.ErrRequiredArgumentMissing, fmt.Sprintf("%v%v", ctv.TXT_MISSING_PARAMETER, ctv.FN_TEMP_DIRECTORY))
			return
		} else {
			tTempDirectory = tempDirectory
		}
		if username == ctv.VAL_EMPTY {
			errorInfo = pi.NewErrorInfo(pi.ErrRequiredArgumentMissing, fmt.Sprintf("%v%v", ctv.TXT_MISSING_PARAMETER, ctv.FN_USERNAME))
			return
		} else {
			tUsername = username
		}
		// environment is validated in awss.NewAWSConfig
		tEnvironment = environment
	} else {
		if tConfigMap, errorInfo = cfgs.GetConfigFile(configFileFQN); errorInfo.Error != nil {
			return
		}
		tSTYHClientId = tConfigMap[ctv.FN_STYH_CLIENT_ID].(string)
		tEnvironment = tConfigMap[ctv.FN_ENVIRONMENT].(string)
		tPassword = tConfigMap[ctv.FN_PASSWORD].(string)
		tConfigMap[ctv.FN_PASSWORD] = ctv.TXT_PROTECTED // Clear the password from memory.
		tSecretKey = tConfigMap[ctv.FN_SECRET_KEY].(string)
		tTempDirectory = tConfigMap[ctv.FN_TEMP_DIRECTORY].(string)
		tUsername = tConfigMap[ctv.FN_USERNAME].(string)
	}

	if errorInfo = validateConfiguration(tSTYHClientId, tEnvironment, tSecretKey, tTempDirectory, tUsername, &tPassword); errorInfo.Error != nil {
		pi.PrintErrorInfo(errorInfo)
		return
	}

	if ai2cClientPtr.awsSettings, errorInfo = awss.LoadAWSCustomerSettings(tEnvironment); errorInfo.Error != nil {
		pi.PrintErrorInfo(errorInfo)
		return
	}
	ai2cClientPtr.environment = tEnvironment
	ai2cClientPtr.tempDirectory = tTempDirectory

	// This returns information about the STYH Customer
	if ai2cClientPtr.styhCustomerConfig.tokens.Access,
		ai2cClientPtr.styhCustomerConfig.tokens.ID,
		ai2cClientPtr.styhCustomerConfig.tokens.Refresh, errorInfo = awss.Login(
		ctv.AUTH_USER_SRP, tUsername, &tPassword,
		ai2cClientPtr.awsSettings.STYHCognitoIdentityInfo, ai2cClientPtr.awsSettings.BaseConfig,
	); errorInfo.Error != nil {
		pi.PrintErrorInfo(errorInfo)
		return
	}

	ai2cClientPtr.styhCustomerConfig.clientId = tSTYHClientId
	ai2cClientPtr.styhCustomerConfig.username = tUsername
	ai2cClientPtr.secretKey = tSecretKey
	tPassword = ctv.TXT_PROTECTED  // Clear the password from memory.
	secretKey = ctv.TXT_PROTECTED  // Clear the secret key from memory.
	tSecretKey = ctv.TXT_PROTECTED // Clear the secret key from memory.

	if errorInfo = processAWSClientParameters(
		ai2cClientPtr.awsSettings,
		ai2cClientPtr.styhCustomerConfig.tokens.ID,
		tEnvironment,
		&ai2cClientPtr.natsConfig,
	); errorInfo.Error != nil {
		pi.PrintErrorInfo(errorInfo)
		return
	}

	if errorInfo = ns.BuildTemporaryFiles(ai2cClientPtr.tempDirectory, ai2cClientPtr.natsConfig); errorInfo.Error != nil {
		pi.PrintErrorInfo(errorInfo)
		return
	}
	ai2cClientPtr.natsConfig.NATSCredentialsFilename = fmt.Sprintf("%v/%v", tTempDirectory, ns.CREDENTIAL_FILENAME)

	if errorInfo = jwts.BuildTLSTemporaryFiles(ai2cClientPtr.tempDirectory, ai2cClientPtr.natsConfig.NATSTLSInfo); errorInfo.Error != nil {
		pi.PrintErrorInfo(errorInfo)
		return
	}
	ai2cClientPtr.natsConfig.NATSTLSInfo.TLSCABundleFQN = fmt.Sprintf("%v/%v", tTempDirectory, jwts.TLS_CA_BUNDLE_FILENAME)
	ai2cClientPtr.natsConfig.NATSTLSInfo.TLSCertFQN = fmt.Sprintf("%v/%v", tTempDirectory, jwts.TLS_CERT_FILENAME)
	ai2cClientPtr.natsConfig.NATSTLSInfo.TLSPrivateKeyFQN = fmt.Sprintf("%v/%v", tTempDirectory, jwts.TLS_PRIVATE_KEY_FILENAME)

	if ai2cClientPtr.natsService.InstanceName, errorInfo = ns.BuildInstanceName(ns.METHOD_DASHES, ai2cClientPtr.styhCustomerConfig.clientId); errorInfo.Error != nil {
		pi.PrintErrorInfo(errorInfo)
		return
	}
	if ai2cClientPtr.natsService.ConnPtr, errorInfo = ns.GetConnection(ai2cClientPtr.natsService.InstanceName, ai2cClientPtr.natsConfig); errorInfo.Error != nil {
		pi.PrintErrorInfo(errorInfo)
		return
	}

	return
}

// AI2PaymentRequest - handles all payment requests. The SaaS providers public or secret key must be provided.
//
// **Cancelling a payment**
// CancellationReason in ai2CPaymentInfo specifies the reason for cancellation.
// The SaaSKey and PaymentIntentId in ai2CPaymentInfo are used to identify the payment to cancel.
// If ai2CPaymentInfo.Keys.Public is empty, ai2CPaymentInfo.Keys.Secret is used as the SaaSKey.
//
// **List Payments**
// List payment intents based on the ReturnRecordsLimit which must be set to a value between 1 and 100.
// Providing the CustomerId will only return payments for that customer. The StartingAfterRecord is the
// pointer where the list return the next record up to the limit.
//
// **List Payment Methods**
// List payment methods is requested when the PaymentMethod is set to LIST.
//
// **Create Payment**
// Creates a payment request when positive amount and the currency are provided.
//
// Customer Messages: None
// Errors: None
// Verifications: None
func (ai2cClientPtr *Ai2CClient) AI2PaymentRequest(ai2CPaymentInfo Ai2CPaymentInfo) (
	reply []byte,
	errorInfo pi.ErrorInfo,
) {

	var (
		tReply *nats.Msg
	)

	if ai2CPaymentInfo.Keys.Public == ctv.VAL_EMPTY && ai2CPaymentInfo.Keys.Secret == ctv.VAL_EMPTY {
		errorInfo = pi.NewErrorInfo(pi.ErrRequiredArgumentMissing, fmt.Sprintf("%v and %v %v", ctv.TXT_PUBLIC_KEY, ctv.TXT_SECRET_KEY, ctv.TXT_ARE_MISSING))
		return
	}

	// Determine Request
	//
	// Request is a cancellation
	if len(ai2CPaymentInfo.CancellationReason) > ctv.VAL_ZERO && len(ai2CPaymentInfo.PaymentIntentId) > ctv.VAL_ZERO {
		tReply, errorInfo = processCancelPaymentIntent(
			ai2cClientPtr.styhCustomerConfig.clientId, ai2cClientPtr.secretKey, &ai2cClientPtr.natsService,
			ai2CPaymentInfo,
		)
		reply = tReply.Data
		return
	}
	// Request is to list payment intents
	if ai2CPaymentInfo.ReturnRecordsLimit > ctv.VAL_ZERO {
		tReply, errorInfo = processListPaymentIntent(
			ai2cClientPtr.styhCustomerConfig.clientId, ai2cClientPtr.secretKey, ai2cClientPtr.styhCustomerConfig.username,
			&ai2cClientPtr.natsService, ai2CPaymentInfo,
		)
		reply = tReply.Data
		return
	}
	// Request is to list payment methods
	if strings.ToLower(ai2CPaymentInfo.PaymentMethod) == ctv.PAYMENT_METHOD_LIST {
		tReply, errorInfo = processListPaymentMethod(ai2cClientPtr.styhCustomerConfig.clientId, ai2cClientPtr.secretKey, &ai2cClientPtr.natsService, ai2CPaymentInfo)
		reply = tReply.Data
		return
	}
	// Request is to create a payment
	if ai2CPaymentInfo.Amount > 0 && len(ai2CPaymentInfo.Currency) > ctv.VAL_ZERO {
		tReply, errorInfo = processCreatePaymentIntent(ai2cClientPtr.styhCustomerConfig.clientId, ai2cClientPtr.secretKey, &ai2cClientPtr.natsService, ai2CPaymentInfo)
		reply = tReply.Data
		return
	}
	// // Request is to confirm a payment
	// if ai2CPaymentInfo.Amount > 0 && len(ai2CPaymentInfo.Currency) > ctv.VAL_ZERO && len(ai2CPaymentInfo.PaymentIntentId) > ctv.VAL_ZERO {
	// 	tReply, errorInfo = (ai2cClientPtr.clientId, &ai2cClientPtr.natsService, ai2CPaymentInfo)
	// 	reply = tReply.Data
	// 	return
	// }

	return
}

// Private Function below here

// processAWSClientParameters - handles getting and storing the shared AWS SSM Parameters.
//
//	Customer Messages: None
//	Errors: None
//	Verifications: None
func processAWSClientParameters(
	awsSettings awss.AWSSettings,
	idToken string,
	environment string,
	natsConfigPtr *ns.NATSConfiguration,
) (errorInfo pi.ErrorInfo) {

	var (
		tParameterName    string
		tParametersOutput awsSSM.GetParametersOutput
		tParameterValue   string
	)

	if tParametersOutput, errorInfo = awss.GetParameters(
		awsSettings.STYHCognitoIdentityInfo,
		awsSettings.BaseConfig,
		idToken,
		ctv.GetParameterName(AI2C_SSM_PARAMETER_PREFIX, environment, ctv.PARAMETER_NATS_TOKEN),
		ctv.GetParameterName(AI2C_SSM_PARAMETER_PREFIX, environment, ctv.PARAMETER_NATS_PORT),
		ctv.GetParameterName(AI2C_SSM_PARAMETER_PREFIX, environment, ctv.PARAMETER_NATS_URL),
		ctv.GetParameterName(AI2C_SSM_PARAMETER_PREFIX, environment, ctv.PARAMETER_TLS_CERT),
		ctv.GetParameterName(AI2C_SSM_PARAMETER_PREFIX, environment, ctv.PARAMETER_TLS_PRIVATE_KEY),
		ctv.GetParameterName(AI2C_SSM_PARAMETER_PREFIX, environment, ctv.PARAMETER_TLS_CA_BUNDLE),
	); errorInfo.Error != nil {
		pi.PrintErrorInfo(errorInfo)
		return
	}

	for _, parameter := range tParametersOutput.Parameters {
		tParameterName = *parameter.Name
		tParameterValue = *parameter.Value
		switch tParameterName {
		case ctv.GetParameterName(AI2C_SSM_PARAMETER_PREFIX, environment, ctv.PARAMETER_NATS_TOKEN):
			natsConfigPtr.NATSToken = tParameterValue
		case ctv.GetParameterName(AI2C_SSM_PARAMETER_PREFIX, environment, ctv.PARAMETER_NATS_PORT):
			natsConfigPtr.NATSPort, _ = strconv.Atoi(tParameterValue)
		case ctv.GetParameterName(AI2C_SSM_PARAMETER_PREFIX, environment, ctv.PARAMETER_NATS_URL):
			natsConfigPtr.NATSURL = tParameterValue
		case ctv.GetParameterName(AI2C_SSM_PARAMETER_PREFIX, environment, ctv.PARAMETER_TLS_CERT):
			natsConfigPtr.NATSTLSInfo.TLSCert = tParameterValue
		case ctv.GetParameterName(AI2C_SSM_PARAMETER_PREFIX, environment, ctv.PARAMETER_TLS_PRIVATE_KEY):
			natsConfigPtr.NATSTLSInfo.TLSPrivateKey = tParameterValue
		case ctv.GetParameterName(AI2C_SSM_PARAMETER_PREFIX, environment, ctv.PARAMETER_TLS_CA_BUNDLE):
			natsConfigPtr.NATSTLSInfo.TLSCABundle = tParameterValue
		default:
			// Optional: Handle unknown parameter names (log a warning?)
		}
	}

	return
}

// processCancelPaymentIntent - handles cancelling a payment intent by sending a request to the NATS service.
// CancellationReason in ai2CPaymentInfo specifies the reason for cancellation.
// The SaaSKey and PaymentIntentId in ai2CPaymentInfo are used to identify the payment intent to cancel.
// If ai2CPaymentInfo.Keys.Public is empty, ai2CPaymentInfo.Keys.Secret is used as the SaaSKey.
// The cancellation request is serialized to JSON and sent as a NATS message with appropriate headers.
//
//	Customer Messages: None
//	Errors: None
//	Verifications: None
func processCancelPaymentIntent(
	clientId, secretKey string,
	natsServicePtr *ns.NATSService,
	ai2CPaymentInfo Ai2CPaymentInfo,
) (
	reply *nats.Msg,
	errorInfo pi.ErrorInfo,
) {

	var (
		tEncryptedRequestData string
		tFunction, _, _, _    = runtime.Caller(0)
		tFunctionName         = runtime.FuncForPC(tFunction).Name()
		tKey                  string
		tNATSHeader           = make(map[string][]string)
		tPIR                  CancelPaymentIntentRequest
		tRequestData          []byte
		tRequestMsg           nats.Msg
	)

	if ai2CPaymentInfo.Keys.Public == ctv.VAL_EMPTY {
		tKey = ai2CPaymentInfo.Keys.Secret
	} else {
		tKey = ai2CPaymentInfo.Keys.Public
	}

	tPIR = CancelPaymentIntentRequest{
		SaaSKey:            tKey,
		PaymentIntentId:    ai2CPaymentInfo.PaymentIntentId,
		CancellationReason: ai2CPaymentInfo.CancellationReason,
	}
	if tRequestData, errorInfo.Error = json.Marshal(tPIR); errorInfo.Error != nil {
		errorInfo = pi.NewErrorInfo(errorInfo.Error, fmt.Sprintf("%v%v - %v%v", ctv.TXT_FUNCTION_NAME, tFunctionName, ctv.TXT_SUBJECT, ctv.SUB_STRIPE_CANCEL_PAYMENT_INTENT))
		return
	}
	if tEncryptedRequestData, errorInfo = jwts.Encrypt(clientId, secretKey, string(tRequestData)); errorInfo.Error != nil {
		return
	}

	tNATSHeader[ctv.FN_CLIENT_ID] = []string{clientId}

	tRequestMsg = nats.Msg{
		Subject: ctv.SUB_STRIPE_CANCEL_PAYMENT_INTENT,
		Header:  tNATSHeader,
		Data:    []byte(tEncryptedRequestData),
	}

	reply, errorInfo = ns.RequestWithHeader(natsServicePtr.ConnPtr, natsServicePtr.InstanceName, &tRequestMsg, 2*time.Second)

	return
}

// processCreatePaymentIntent - handles creating a payment intent request and sending it to the NATS service.
//
// Customer Messages: None
// Errors: None
// Verifications: None
func processCreatePaymentIntent(
	clientId, secretKey string,
	natsServicePtr *ns.NATSService,
	ai2CPaymentInfo Ai2CPaymentInfo,
) (
	reply *nats.Msg,
	errorInfo pi.ErrorInfo,
) {

	var (
		tEncryptedRequestData string
		tFunction, _, _, _    = runtime.Caller(0)
		tFunctionName         = runtime.FuncForPC(tFunction).Name()
		tKey                  string
		tNATSHeader           = make(map[string][]string)
		tPIR                  PaymentIntentRequest
		tRequestData          []byte
		tRequestMsg           nats.Msg
	)

	if ai2CPaymentInfo.Keys.Public == ctv.VAL_EMPTY {
		tKey = ai2CPaymentInfo.Keys.Secret
	} else {
		tKey = ai2CPaymentInfo.Keys.Public
	}

	tPIR = PaymentIntentRequest{
		Amount:                  ai2CPaymentInfo.Amount,
		AutomaticPaymentMethods: ai2CPaymentInfo.UseAutomaticPaymentMethod,
		Currency:                ai2CPaymentInfo.Currency,
		Description:             ai2CPaymentInfo.Description,
		ReceiptEmail:            ai2CPaymentInfo.ReceiptEmail,
		ReturnURL:               ai2CPaymentInfo.ReturnURL,
		SaaSKey:                 tKey,
	}

	if tRequestData, errorInfo.Error = json.Marshal(tPIR); errorInfo.Error != nil {
		errorInfo = pi.NewErrorInfo(errorInfo.Error, fmt.Sprintf("%v%v - %v%v", ctv.TXT_FUNCTION_NAME, tFunctionName, ctv.TXT_SUBJECT, ctv.SUB_STRIPE_CREATE_PAYMENT_INTENT))
		return
	}
	if tEncryptedRequestData, errorInfo = jwts.Encrypt(clientId, secretKey, string(tRequestData)); errorInfo.Error != nil {
		return
	}

	tNATSHeader[ctv.FN_CLIENT_ID] = []string{clientId}

	tRequestMsg = nats.Msg{
		Subject: ctv.SUB_STRIPE_CREATE_PAYMENT_INTENT,
		Header:  tNATSHeader,
		Data:    []byte(tEncryptedRequestData),
	}

	reply, errorInfo = ns.RequestWithHeader(natsServicePtr.ConnPtr, natsServicePtr.InstanceName, &tRequestMsg, 2*time.Second)

	return
}

// processListPaymentIntent - sends a NATS request to list payment intents based on the provided criteria.
// The ReturnRecordsLimit must be set to a value between 1 and 100.
//
//	Customer Messages: None
//	Errors: None
//	Verifications: None
func processListPaymentIntent(
	clientId, secretKey, username string,
	natsServicePtr *ns.NATSService,
	ai2CPaymentInfo Ai2CPaymentInfo,
) (
	reply *nats.Msg,
	errorInfo pi.ErrorInfo,
) {

	var (
		tEncryptedRequestData string
		tFunction, _, _, _    = runtime.Caller(0)
		tFunctionName         = runtime.FuncForPC(tFunction).Name()
		tKey                  string
		tNATSHeader           = make(map[string][]string)
		tPIR                  ListPaymentIntentRequest
		tRequestData          []byte
		tRequestMsg           nats.Msg
	)

	if ai2CPaymentInfo.Keys.Public == ctv.VAL_EMPTY {
		tKey = ai2CPaymentInfo.Keys.Secret
	} else {
		tKey = ai2CPaymentInfo.Keys.Public
	}

	tPIR = ListPaymentIntentRequest{
		SaaSKey:       tKey,
		CustomerId:    ai2CPaymentInfo.CustomerId,
		Limit:         ai2CPaymentInfo.ReturnRecordsLimit,
		StartingAfter: ai2CPaymentInfo.StartingAfterRecord,
	}
	if tRequestData, errorInfo.Error = json.Marshal(tPIR); errorInfo.Error != nil {
		errorInfo = pi.NewErrorInfo(errorInfo.Error, fmt.Sprintf("%v%v - %v%v", ctv.TXT_FUNCTION_NAME, tFunctionName, ctv.TXT_SUBJECT, ctv.SUB_STRIPE_LIST_PAYMENT_INTENTS))
		return
	}
	if tEncryptedRequestData, errorInfo = jwts.Encrypt(clientId, secretKey, string(tRequestData)); errorInfo.Error != nil {
		return
	}

	tNATSHeader[ctv.FN_CLIENT_ID] = []string{clientId}
	tNATSHeader[ctv.FN_USERNAME] = []string{username}

	tRequestMsg = nats.Msg{
		Subject: ctv.SUB_STRIPE_LIST_PAYMENT_INTENTS,
		Header:  tNATSHeader,
		Data:    []byte(tEncryptedRequestData),
	}

	reply, errorInfo = ns.RequestWithHeader(natsServicePtr.ConnPtr, natsServicePtr.InstanceName, &tRequestMsg, 2*time.Second)

	return
}

// processListPaymentMethod - sends a request to list payment methods to the NATS service using the Stripe
// subject. It marshals the request payload and adds the client ID to the request header.
//
//	Customer Messages: None
//	Errors: None
//	Verifications: None
func processListPaymentMethod(
	clientId, secretKey string,
	natsServicePtr *ns.NATSService,
	ai2CPaymentInfo Ai2CPaymentInfo,
) (
	reply *nats.Msg,
	errorInfo pi.ErrorInfo,
) {

	var (
		tEncryptedRequestData string
		tFunction, _, _, _    = runtime.Caller(0)
		tFunctionName         = runtime.FuncForPC(tFunction).Name()
		tKey                  string
		tNATSHeader           = make(map[string][]string)
		tPIR                  ListPaymentMethodRequest
		tRequestData          []byte
		tRequestMsg           nats.Msg
	)

	if ai2CPaymentInfo.Keys.Public == ctv.VAL_EMPTY {
		tKey = ai2CPaymentInfo.Keys.Secret
	} else {
		tKey = ai2CPaymentInfo.Keys.Public
	}

	tPIR = ListPaymentMethodRequest{
		SaaSKey: tKey,
	}
	if tRequestData, errorInfo.Error = json.Marshal(tPIR); errorInfo.Error != nil {
		errorInfo = pi.NewErrorInfo(errorInfo.Error, fmt.Sprintf("%v%v - %v%v", ctv.TXT_FUNCTION_NAME, tFunctionName, ctv.TXT_SUBJECT, ctv.SUB_STRIPE_LIST_PAYMENT_METHODS))
		return
	}
	if tEncryptedRequestData, errorInfo = jwts.Encrypt(clientId, secretKey, string(tRequestData)); errorInfo.Error != nil {
		return
	}

	tNATSHeader[ctv.FN_CLIENT_ID] = []string{clientId}

	tRequestMsg = nats.Msg{
		Subject: ctv.SUB_STRIPE_LIST_PAYMENT_METHODS,
		Header:  tNATSHeader,
		Data:    []byte(tEncryptedRequestData),
	}

	reply, errorInfo = ns.RequestWithHeader(natsServicePtr.ConnPtr, natsServicePtr.InstanceName, &tRequestMsg, 2*time.Second)

	return
}

// validateConfiguration - checks the values in the configuration file are valid. ValidateConfiguration doesn't
// test if the configuration file exists, readable, or parsable.
//
//	Customer Messages: None
//	Errors: ErrEnvironmentInvalid, ErrRequiredArgumentMissing
//	Verifications: None
func validateConfiguration(
	styhClientId, environment, secretKey, tempDirectory, username string,
	passwordPtr *string,
) (
	errorInfo pi.ErrorInfo,
) {

	if styhClientId == ctv.VAL_EMPTY {
		errorInfo = pi.NewErrorInfo(pi.ErrRequiredArgumentMissing, fmt.Sprintf("%v%v", ctv.TXT_MISSING_PARAMETER, ctv.FN_STYH_CLIENT_ID))
		return
	}
	if hv.IsEnvironmentValid(environment) == false {
		errorInfo = pi.NewErrorInfo(pi.ErrEnvironmentInvalid, fmt.Sprintf("%v%v", ctv.TXT_EVIRONMENT, ctv.FN_ENVIRONMENT))
		return
	}
	if passwordPtr == nil {
		errorInfo = pi.NewErrorInfo(pi.ErrRequiredArgumentMissing, fmt.Sprintf("%v%v", ctv.TXT_MISSING_PARAMETER, ctv.FN_PASSWORD))
		return
	}
	if secretKey == ctv.VAL_EMPTY {
		errorInfo = pi.NewErrorInfo(pi.ErrRequiredArgumentMissing, fmt.Sprintf("%v%v", ctv.TXT_MISSING_PARAMETER, ctv.FN_SECRET_KEY))
		return
	}
	if tempDirectory == ctv.VAL_EMPTY {
		errorInfo = pi.NewErrorInfo(pi.ErrRequiredArgumentMissing, fmt.Sprintf("%v%v", ctv.TXT_MISSING_PARAMETER, ctv.FN_TEMP_DIRECTORY))
		return
	}
	if username == ctv.VAL_EMPTY {
		errorInfo = pi.NewErrorInfo(pi.ErrRequiredArgumentMissing, fmt.Sprintf("%v%v", ctv.TXT_MISSING_PARAMETER, ctv.FN_USERNAME))
		return
	}

	return
}
