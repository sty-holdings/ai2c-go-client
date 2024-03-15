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

	ctv "github.com/sty-holdings/constant-type-vars-go/v2024"
	awss "github.com/sty-holdings/sty-shared/v2024/awsServices"
	cfgs "github.com/sty-holdings/sty-shared/v2024/configuration"
	pi "github.com/sty-holdings/sty-shared/v2024/programInfo"
)

func NewAI2Client(username, password, clientId, secretKey, environment, configFileFQN string) (
	awssPtr *awss.AWSSession,
	errorInfo pi.ErrorInfo,
) {

	var (
		// config       Ai2ClientConfig
		tEnvironment string
		tPassword    string
		tUsername    string
	)

	var (
		tConfigMap = make(map[string]interface{})
	)

	if configFileFQN == ctv.VAL_EMPTY {
		if clientId == ctv.VAL_EMPTY {
			errorInfo = pi.NewErrorInfo(pi.ErrRequiredArgumentMissing, fmt.Sprintf("%v%v", ctv.TXT_MISSING_PARAMETER, ctv.FN_CLIENT_ID))
			return
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
		fmt.Println(tConfigMap)
		tEnvironment = tConfigMap["environment"].(string)
		tPassword = tConfigMap["password"].(string)
		tConfigMap["password"] = ctv.TXT_PROTECTED // Clear the password from memory.
		tUsername = tConfigMap["username"].(string)
	}

	if awssPtr, errorInfo = awss.NewAWSConfig(tEnvironment); errorInfo.Error != nil {
		pi.PrintErrorInfo(errorInfo)
		return
	}
	if _, errorInfo = awss.Login(ctv.AUTH_USER_SRP, tUsername, &tPassword, awssPtr); errorInfo.Error != nil {
		pi.PrintErrorInfo(errorInfo)
		return
	}

	tPassword = ctv.TXT_PROTECTED // Clear the password from memory.

	return
	// Set up goes here
}
