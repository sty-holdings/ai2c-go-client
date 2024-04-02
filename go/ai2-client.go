// Package main
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
package main

import (
	"fmt"
	"strconv"

	awsSSM "github.com/aws/aws-sdk-go-v2/service/ssm"
	ctv "github.com/sty-holdings/constant-type-vars-go/v2024"
	awss "github.com/sty-holdings/sty-shared/v2024/awsServices"
	cfgs "github.com/sty-holdings/sty-shared/v2024/configuration"
	hv "github.com/sty-holdings/sty-shared/v2024/helpersValidators"
	jwts "github.com/sty-holdings/sty-shared/v2024/jwtServices"
	ns "github.com/sty-holdings/sty-shared/v2024/natsSerices"
	pi "github.com/sty-holdings/sty-shared/v2024/programInfo"
)

func NewAI2Client(clientId, environment, password, secretKey, tempDirectory, username, configFileFQN string) (
	ai2ClientPtr Ai2Client,
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

	if ai2ClientPtr.awssPtr, errorInfo = awss.NewAWSConfig(tEnvironment); errorInfo.Error != nil {
		pi.PrintErrorInfo(errorInfo)
		return
	}
	ai2ClientPtr.environment = tEnvironment
	ai2ClientPtr.tempDirectory = tTempDirectory

	if errorInfo = awss.Login(ctv.AUTH_USER_SRP, tUsername, &tPassword, ai2ClientPtr.awssPtr); errorInfo.Error != nil {
		pi.PrintErrorInfo(errorInfo)
		return
	}

	ai2ClientPtr.secretKey = tSecretKey
	tPassword = ctv.TXT_PROTECTED  // Clear the password from memory.
	secretKey = ctv.TXT_PROTECTED  // Clear the secret key from memory.
	tSecretKey = ctv.TXT_PROTECTED // Clear the secret key from memory.

	if errorInfo = processAWSClientParameters(ai2ClientPtr.awssPtr, tEnvironment, &ai2ClientPtr.natsConfig); errorInfo.Error != nil {
		pi.PrintErrorInfo(errorInfo)
		return
	}

	if errorInfo = ns.BuildTemporaryFiles(ai2ClientPtr.tempDirectory, ai2ClientPtr.natsConfig); errorInfo.Error != nil {
		pi.PrintErrorInfo(errorInfo)
		return
	}
	ai2ClientPtr.natsConfig.NATSCredentialsFilename = fmt.Sprintf("%v/%v", tTempDirectory, ns.CREDENTIAL_FILENAME)

	if errorInfo = jwts.BuildTLSTemporaryFiles(ai2ClientPtr.tempDirectory, ai2ClientPtr.natsConfig.NATSTLSInfo); errorInfo.Error != nil {
		pi.PrintErrorInfo(errorInfo)
		return
	}
	ai2ClientPtr.natsConfig.NATSTLSInfo.TLSCABundleFQN = fmt.Sprintf("%v/%v", tTempDirectory, jwts.TLS_CA_BUNDLE_FILENAME)
	ai2ClientPtr.natsConfig.NATSTLSInfo.TLSCertFQN = fmt.Sprintf("%v/%v", tTempDirectory, jwts.TLS_CERT_FILENAME)
	ai2ClientPtr.natsConfig.NATSTLSInfo.TLSPrivateKeyFQN = fmt.Sprintf("%v/%v", tTempDirectory, jwts.TLS_PRIVATE_KEY_FILENAME)

	if ai2ClientPtr.natsService.InstanceName, errorInfo = ns.BuildInstanceName(ns.METHOD_DASHES, awss.GetClientId(ai2ClientPtr.awssPtr)); errorInfo.Error != nil {
		pi.PrintErrorInfo(errorInfo)
		return
	}
	if ai2ClientPtr.natsService.ConnPtr, errorInfo = ns.GetConnection(ai2ClientPtr.natsService.InstanceName, ai2ClientPtr.natsConfig); errorInfo.Error != nil {
		pi.PrintErrorInfo(errorInfo)
		return
	}

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
		ctv.GetParameterName(ctv.PARAMETER_NATS_TOKEN, PROGRAM_NAME, environment),
		ctv.GetParameterName(ctv.PARAMETER_NATS_PORT, PROGRAM_NAME, environment),
		ctv.GetParameterName(ctv.PARAMETER_NATS_URL, PROGRAM_NAME, environment),
		ctv.GetParameterName(ctv.PARAMETER_TLS_CERT, PROGRAM_NAME, environment),
		ctv.GetParameterName(ctv.PARAMETER_TLS_PRIVATE_KEY, PROGRAM_NAME, environment),
		ctv.GetParameterName(ctv.PARAMETER_TLS_CA_BUNDLE, PROGRAM_NAME, environment),
	); errorInfo.Error != nil {
		pi.PrintErrorInfo(errorInfo)
		return
	}

	for _, parameter := range tParametersOutput.Parameters {
		tParameterName = *parameter.Name
		tParameterValue = *parameter.Value
		if tParameterName == ctv.GetParameterName(ctv.PARAMETER_NATS_TOKEN, PROGRAM_NAME, environment) {
			natsConfigPtr.NATSToken = tParameterValue
		}
		if tParameterName == ctv.GetParameterName(ctv.PARAMETER_NATS_PORT, PROGRAM_NAME, environment) {
			natsConfigPtr.NATSPort, _ = strconv.Atoi(tParameterValue)
		}
		if tParameterName == ctv.GetParameterName(ctv.PARAMETER_NATS_URL, PROGRAM_NAME, environment) {
			natsConfigPtr.NATSURL = tParameterValue
		}
		if tParameterName == ctv.GetParameterName(ctv.PARAMETER_TLS_CERT, PROGRAM_NAME, environment) {
			natsConfigPtr.NATSTLSInfo.TLSCert = tParameterValue
		}
		if tParameterName == ctv.GetParameterName(ctv.PARAMETER_TLS_PRIVATE_KEY, PROGRAM_NAME, environment) {
			natsConfigPtr.NATSTLSInfo.TLSPrivateKey = tParameterValue
		}
		if tParameterName == ctv.GetParameterName(ctv.PARAMETER_TLS_CA_BUNDLE, PROGRAM_NAME, environment) {
			natsConfigPtr.NATSTLSInfo.TLSCABundle = tParameterValue
		}
	}

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
