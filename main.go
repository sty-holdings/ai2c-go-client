// Package main
/*
====> This is a sample usage of NATS Connect. The CLI part is to allow an easy place to start.
====> The run function is the code to drop into you program.


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
	"log"
	"os"

	"ai2c-go-client/src"
	"github.com/integrii/flaggy"
	ctv "github.com/sty-holdings/constant-type-vars-go/v2024"
	config "github.com/sty-holdings/sty-shared/v2024/configuration"
	pi "github.com/sty-holdings/sty-shared/v2024/programInfo"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

var (
	clientId       string
	configFileFQN  string
	generateConfig bool
	password       string
	environment    = "production" // this is the default. For development, use 'development'.
	programName    = "Ai2C-go-client"
	secretKey      string
	tempDirectory  string
	testingOn      bool
	username       string
	version        = "9999.9999.9999"
)

func init() {

	appDescription := cases.Title(language.English).String(programName) + " is a driver program to test the NATS Connect Stripe integration." +
		"\nVersion: \n" +
		ctv.SPACES_FOUR + "- " + version + "\n" +
		"\nConstraints: \n" +
		ctv.SPACES_FOUR + "- When using -c you must pass the fully qualified configuration file name.\n" +
		ctv.SPACES_FOUR + "- There is no console available at this time and all log messages are output to your Log_Directory " +
		"\n\tspecified in the config file or command line below.\n" +
		"\nNotes:\n" +
		ctv.SPACES_FOUR + "None\n" +
		"\nFor more info, see link below:\n"

	// Set your program's name and description.  These appear in help output.
	flaggy.SetName("\n" + programName) // "\n" is added to the start of the name to make the output easier to read.
	flaggy.SetDescription(appDescription)

	// You can disable various things by changing bool on the default parser
	// (or your own parser if you have created one).
	flaggy.DefaultParser.ShowHelpOnUnexpected = true

	// You can set a help prepend or append on the default parser.
	flaggy.DefaultParser.AdditionalHelpPrepend = "https://github.com/styh-dev/albert"

	// Add a flag to the main program (this will be available in all subcommands as well).
	flaggy.String(&configFileFQN, "c", "config", "Provides the setup information needed by and is required to start the program.")
	flaggy.Bool(&generateConfig, "gc", "genconfig", "This will output a skeleton configuration and note files.\n\t\t\tThis will cause all other options to be ignored.")
	flaggy.String(&clientId, "ci", "clientId", "The AI2 Connect assigned client id. You can find it here: https://production-nc-dashboard.web.app/.")
	flaggy.String(&password, "p", "password", "The password you selected when you signed up for AI2 connect services. This is encrypted using SSL and only exist in Cognito.")
	flaggy.String(
		&secretKey, "sk", "secretKey", "The AI2 Connect assigned secret key. This is encrypted using SSL and a new can be generated at https://production-nc-dashboard."+
			"web.app/.",
	)
	flaggy.String(
		&tempDirectory, "tmp", "tempDir", "The temporary directory where the Ai2 Client can read and write temporary files.",
	)
	flaggy.Bool(&testingOn, "t", "testingOn", "This puts the program into testing mode.")
	flaggy.String(&username, "u", "username", "The username you selected when you signed up for AI2 connect services. This is encrypted using SSL and only exist in Cognito.")

	// Set the version and parse all inputs into variables.
	flaggy.SetVersion(version)
	flaggy.Parse()
}

func main() {

	fmt.Println()
	log.Printf("Running %v.\n", programName)

	if generateConfig {
		config.GenerateConfigFileSkeleton(programName, "config/")
		os.Exit(0)
	}

	// This is to prevent the serverName from being empty.
	if programName == ctv.VAL_EMPTY {
		pi.PrintError(pi.ErrProgramNameMissing, fmt.Sprintf("%v %v", ctv.TXT_PROGRAM_NAME, programName))
		os.Exit(1)
	}

	if testingOn == false {
		// This is to prevent the version from being empty or not being set during the build process.
		if version == ctv.VAL_EMPTY || version == "9999.9999.9999" {
			pi.PrintError(pi.ErrVersionInvalid, fmt.Sprintf("%v %v", ctv.TXT_SERVER_VERSION, version))
			flaggy.ShowHelpAndExit("")
		}
		if username == ctv.VAL_EMPTY || password == ctv.VAL_EMPTY || clientId == ctv.VAL_EMPTY || secretKey == ctv.VAL_EMPTY || tempDirectory == ctv.VAL_EMPTY {
			// Has the config file location and name been provided, if not, return help.
			if configFileFQN == "" || configFileFQN == "-t" {
				flaggy.ShowHelpAndExit("")
			}
		}
	}

	run(clientId, environment, password, secretKey, tempDirectory, username, configFileFQN)

	os.Exit(0)
}

func run(clientId, environment, password, secretKey, tempDirectory, username, configFileFQN string) {

	var (
		errorInfo pi.ErrorInfo
		clientPtr src.Ai2Client
	)

	if clientPtr, errorInfo = src.NewAI2Client(clientId, environment, password, secretKey, tempDirectory, username, configFileFQN); errorInfo.Error != nil {
		pi.PrintErrorInfo(errorInfo)
		flaggy.ShowHelpAndExit("")
	}

	fmt.Println(clientPtr)
}
