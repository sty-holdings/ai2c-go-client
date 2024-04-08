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
	awssPtr       *awss.AWSSession
	clientId      string
	environment   string
	natsService   ns.NATSService
	natsConfig    ns.NATSConfiguration
	secretKey     string
	tempDirectory string
}

type Ai2CInfo struct {
	Amount                  float64 `json:"amount,omitempty"`
	AutomaticPaymentMethods bool    `json:"automatic_payment_methods,omitempty"`
	CancellationReason      string  `json:"cancellation_reason,omitempty"`
	CaptureMethod           string  `json:"capture_method,omitempty"`
	Currency                string  `json:"currency,omitempty"`
	CustomerId              string  `json:"customer_id,omitempty"`
	Description             string  `json:"description,omitempty"`
	Key                     string  `json:"key"`
	Limit                   int64   `json:"limit,omitempty"`
	PaymentIntentId         string  `json:"id,omitempty"`
	PaymentMethod           string  `json:"payment_method,omitempty"`
	ReceiptEmail            string  `json:"receipt_email,omitempty"`
	ReturnURL               string  `json:"return_url,omitempty,omitempty"`
	StartingAfter           string  `json:"starting_after,omitempty"`
}

type PaymentIntentRequest struct {
	Amount                  float64 `json:"amount"`
	AutomaticPaymentMethods bool    `json:"automatic_payment_methods,omitempty"`
	Currency                string  `json:"currency"`
	Description             string  `json:"description,omitempty"`
	Key                     string  `json:"key"`
	ReceiptEmail            string  `json:"receipt_email"`
	ReturnURL               string  `json:"return_url,omitempty"`
	// Confirm            bool     `json:"confirm,omitempty"`
	// PaymentMethodTypes []string `json:"payment_method_types,omitempty"`
}

func NewAI2CClient(clientId, environment, password, secretKey, tempDirectory, username, configFileFQN string) (
	ai2cClientPtr Ai2CClient,
	errorInfo pi.ErrorInfo,
) {

	var (
		tClientId      string
		tEnvironment   string
		tPassword      string
		tSecretKey     string
		tTempDirectory string
		tUsername      string
	)

	var (
		tConfigMap = make(map[string]interface{})
	)

	if configFileFQN == ctv.VAL_EMPTY {
		if clientId == ctv.VAL_EMPTY {
			errorInfo = pi.NewErrorInfo(pi.ErrRequiredArgumentMissing, fmt.Sprintf("%v%v", ctv.TXT_MISSING_PARAMETER, ctv.FN_CLIENT_ID))
			return
		} else {
			tClientId = clientId
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
		tClientId = tConfigMap[ctv.FN_CLIENT_ID].(string)
		tEnvironment = tConfigMap[ctv.FN_ENVIRONMENT].(string)
		tPassword = tConfigMap[ctv.FN_PASSWORD].(string)
		tConfigMap[ctv.FN_PASSWORD] = ctv.TXT_PROTECTED // Clear the password from memory.
		tSecretKey = tConfigMap[ctv.FN_SECRET_KEY].(string)
		tTempDirectory = tConfigMap[ctv.FN_TEMP_DIRECTORY].(string)
		tUsername = tConfigMap[ctv.FN_USERNAME].(string)
	}

	if errorInfo = validateConfiguration(tClientId, tEnvironment, tSecretKey, tTempDirectory, tUsername, &tPassword); errorInfo.Error != nil {
		pi.PrintErrorInfo(errorInfo)
		return
	}

	if ai2cClientPtr.awssPtr, errorInfo = awss.NewAWSConfig(tEnvironment); errorInfo.Error != nil {
		pi.PrintErrorInfo(errorInfo)
		return
	}
	ai2cClientPtr.environment = tEnvironment
	ai2cClientPtr.tempDirectory = tTempDirectory
	ai2cClientPtr.clientId = tClientId

	if errorInfo = awss.Login(ctv.AUTH_USER_SRP, tUsername, &tPassword, ai2cClientPtr.awssPtr); errorInfo.Error != nil {
		pi.PrintErrorInfo(errorInfo)
		return
	}

	ai2cClientPtr.secretKey = tSecretKey
	tPassword = ctv.TXT_PROTECTED  // Clear the password from memory.
	secretKey = ctv.TXT_PROTECTED  // Clear the secret key from memory.
	tSecretKey = ctv.TXT_PROTECTED // Clear the secret key from memory.

	if errorInfo = processAWSClientParameters(ai2cClientPtr.awssPtr, tEnvironment, &ai2cClientPtr.natsConfig); errorInfo.Error != nil {
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

	if ai2cClientPtr.natsService.InstanceName, errorInfo = ns.BuildInstanceName(ns.METHOD_DASHES, awss.GetClientId(ai2cClientPtr.awssPtr)); errorInfo.Error != nil {
		pi.PrintErrorInfo(errorInfo)
		return
	}
	if ai2cClientPtr.natsService.ConnPtr, errorInfo = ns.GetConnection(ai2cClientPtr.natsService.InstanceName, ai2cClientPtr.natsConfig); errorInfo.Error != nil {
		pi.PrintErrorInfo(errorInfo)
		return
	}

	return
}

func (ai2cClientPtr *Ai2CClient) AI2Request(ai2cInfo Ai2CInfo) (
	errorInfo pi.ErrorInfo,
) {

	if ai2cInfo.Key <= ctv.VAL_EMPTY {
		errorInfo = pi.NewErrorInfo(pi.ErrRequiredArgumentMissing, fmt.Sprintf("%v%v", ctv.TXT_AI2C_KEY, ctv.FN_KEY))
		return
	}

	if ai2cInfo.Amount > 0 { // Use wants to create a payment
		processCreatePaymentIntent(ai2cClientPtr.clientId, &ai2cClientPtr.natsService, ai2cInfo)
		return
	}

	// if ai2cInfo.CustomerId != ctv.VAL_EMPTY {
	// 	processStripeCustomer(ai2cInfo)
	// 	return
	// }
	//
	// if ai2cInfo.PaymentIntentId != ctv.VAL_EMPTY {
	// 	processStripeCustomer(ai2cInfo)
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
	awssPtr *awss.AWSSession,
	environment string,
	natsConfigPtr *ns.NATSConfiguration,
) (errorInfo pi.ErrorInfo) {

	var (
		tParameterName    string
		tParametersOutput awsSSM.GetParametersOutput
		tParameterValue   string
	)

	if tParametersOutput, errorInfo = awss.GetParameters(
		awssPtr,
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

func processCreatePaymentIntent(
	clientId string,
	natsServicePtr *ns.NATSService,
	ai2cInfo Ai2CInfo,
) (errorInfo pi.ErrorInfo) {

	var (
		tFunction, _, _, _ = runtime.Caller(0)
		tFunctionName      = runtime.FuncForPC(tFunction).Name()
		tPIR               PaymentIntentRequest
		tRequestData       []byte
		tRequestMsg        *nats.Msg
	)

	tPIR = PaymentIntentRequest{
		Amount:                  ai2cInfo.Amount,
		AutomaticPaymentMethods: ai2cInfo.AutomaticPaymentMethods,
		Currency:                ai2cInfo.Currency,
		Description:             ai2cInfo.Description,
		Key:                     "",
		ReceiptEmail:            ai2cInfo.ReceiptEmail,
		ReturnURL:               ai2cInfo.ReturnURL,
	}

	if tRequestData, errorInfo.Error = json.Marshal(tPIR); errorInfo.Error != nil {
		errorInfo = pi.NewErrorInfo(errorInfo.Error, fmt.Sprintf("%v%v - %v%v", ctv.TXT_FUNCTION_NAME, tFunctionName, ctv.TXT_SUBJECT, ctv.SUB_STRIPE_CREATE_PAYMENT_INTENT))
		return
	}

	tRequestMsg.Header.Add(ctv.FN_CLIENT_ID, clientId)
	tRequestMsg.Data = tRequestData
	tRequestMsg.Subject = ctv.SUB_STRIPE_CREATE_PAYMENT_INTENT

	ns.RequestWithHeader(natsServicePtr.ConnPtr, natsServicePtr.InstanceName, tRequestMsg, 2*time.Second)

	return
}

// validateConfiguration - checks the values in the configuration file are valid. ValidateConfiguration doesn't
// test if the configuration file exists, readable, or parsable.
//
//	Customer Messages: None
//	Errors: ErrEnvironmentInvalid, ErrRequiredArgumentMissing
//	Verifications: None
func validateConfiguration(
	clientId, environment, secretKey, tempDirectory, username string,
	passwordPtr *string,
) (
	errorInfo pi.ErrorInfo,
) {

	if clientId == ctv.VAL_EMPTY {
		errorInfo = pi.NewErrorInfo(pi.ErrRequiredArgumentMissing, fmt.Sprintf("%v%v", ctv.TXT_MISSING_PARAMETER, ctv.FN_CLIENT_ID))
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
