// Package main
/*
General description of the purpose of the go file.

RESTRICTIONS:
    AWS functions:
    * Program must have access to a .aws/credentials file in the default location.
    * This will only access system parameters that start with '/sote' (ROOTPATH).
    * {Enter other restrictions here for AWS

    {Other catagories of restrictions}
    * {List of restrictions for the catagory

NOTES:
    {Enter any additional notes that you believe will help the next developer.}

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
	config "github.com/sty-holdings/sty-shared/v2024/configuration"
	pi "github.com/sty-holdings/sty-shared/v2024/programInfo"
)

func stripeRequest(username, password, clientId, secretKey, configFileFQN string) (errorInfo pi.ErrorInfo) {

	var (
		configPtr *Ai2ClientConfig
	)

	if configFileFQN == ctv.VAL_EMPTY {
		if username == ctv.VAL_EMPTY {
			errorInfo = pi.NewErrorInfo(pi.ErrRequiredArgumentMissing, fmt.Sprintf("%v%v", ctv.TXT_MISSING_PARAMETER, ctv.FN_USERNAME))
			return
		}
		if password == ctv.VAL_EMPTY {
			errorInfo = pi.NewErrorInfo(pi.ErrRequiredArgumentMissing, fmt.Sprintf("%v%v", ctv.TXT_MISSING_PARAMETER, ctv.FN_PASSWORD))
			return
		}
		if clientId == ctv.VAL_EMPTY {
			errorInfo = pi.NewErrorInfo(pi.ErrRequiredArgumentMissing, fmt.Sprintf("%v%v", ctv.TXT_MISSING_PARAMETER, ctv.FN_CLIENT_ID))
			return
		}
		if secretKey == ctv.VAL_EMPTY {
			errorInfo = pi.NewErrorInfo(pi.ErrRequiredArgumentMissing, fmt.Sprintf("%v%v", ctv.TXT_MISSING_PARAMETER, ctv.FN_SECRET_KEY))
			return
		}
	} else {
		config.GetConfigFile(configFileFQN, configPtr)
	}

	awsHelper, errorInfo := awss.NewAWSSession()

	return
	// Set up goes here
}