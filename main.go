package main

import (
	"alexa-skill-test/src/alexa"
	"alexa-skill-test/src/countries"
	"alexa-skill-test/src/nationality"
	"alexa-skill-test/src/user"
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/aws/aws-lambda-go/lambda"
)

// HandleHelpIntent handles requests for help from users of the skill
func HandleHelpIntent(request alexa.Request) alexa.Response {
	// builder is used instead of alexa simple response for more
	// sophisticated response including voice pauses and other features
	var builder alexa.SSMLBuilder
	builder.Say("You can ask me like so:")
	builder.Pause("1000")
	builder.Say("My name is Ethan, where am I from?")
	return alexa.NewSSMLResponse("Help", builder.Build())
}

// HandleAboutIntent handles requests from users asking about the skill
func HandleAboutIntent(request alexa.Request) alexa.Response {
	// NewSimpleResponse responds with simple text to the client using the skill
	return alexa.NewSimpleResponse("About", "Thanks for using me! I can guess your nationality based on your first name. After providing me with your name, I'll list some countries where you might be from, along with a probability for each of them!")
}

// HandleGuessIntent is the most important handler.
// It resolves any request asking for the main feature
// of the skill which is guessing what nationality is the
// person based on their name that they provided with the request.
// A user can say:
// Alexa, ask nationality guesser to guess my nationality, my name is Ethan
func HandleGuessIntent(request alexa.Request, usingLinkedAccount bool) alexa.Response {
	var firstName string
	if usingLinkedAccount {
		// get name using user's linked account
		firstName = fetchGivenName(request.Session.User.AccessToken)
	} else {
		// extract first name of user from the request slots
		firstName = getValueOfName(request.Body.Intent.Slots, "first_name")
	}

	fmt.Println(firstName)

	// fetch nationality guesses from the network for the name extracted above
	// the API returns country codes for which the person might be from
	predictionsResponse := fetchNationalityPredictions(firstName)

	// append all country codes to an array of codes
	countryCodes := appendCountryCodes(predictionsResponse)

	// Using country codes we have,
	// fetch information about those countries from the network
	countries := fetchCountriesOfCodes(countryCodes)

	// Build and send response using data above
	response := buildGuessResponse(countries, predictionsResponse)
	return alexa.NewSSMLResponse("Nationality Guess", response)
}

// API sending nationality guesses returns country codes for guesses
// appendCountryCodes takes a response object and returns an array of country codes
// included in that response
func appendCountryCodes(response nationality.Response) []string {
	// append all country codes to an array of codes
	var countryCodes []string
	for _, v := range response.Predictions {
		countryCodes = append(countryCodes, v.Country_id)
	}
	return countryCodes
}

// buildGuessResponse creates a response builder and builds a guessing
// response to be sent to the skill user
func buildGuessResponse(countries countries.Country, predictionsResponse nationality.Response) string {
	// Build and send response using data above
	var builder alexa.SSMLBuilder

	if len(predictionsResponse.Predictions) == 0 {
		// If no guesses are found for the name provided, return a message
		builder.Say(fmt.Sprintf("Sorry, I couldn't guess your nationality based on the name you provided. Try again with your friends' names!"))
	} else {
		builder.Say("There is a")
		// Otherwise, loop through guesses
		for i, v := range predictionsResponse.Predictions {
			// if it's the first guess, don't pause before saying it, otherwise do.
			if i != 0 {
				builder.Pause("500")
			}
			// Use information fetched to say a guess with a probability and a demonym
			builder.Say(fmt.Sprintf("%d percent chance you're %s.", int(v.Probability*100), findCountryOfCode(countries, v.Country_id)))
		}
	}
	return builder.Build()
}

// Given a list of country struct objects
// findCountryOfCode finds the country having a specific code
// and returns the Demonym of that country/nationality
func findCountryOfCode(countries countries.Country, code string) string {
	for _, v := range countries {
		if v.Code == code {
			return v.Demonym
		}
	}
	return "Unknown"
}

// Given slots received with the request
// getValueOfName returns slot value of the slot
// having the struct field "Name" value equal to the string parameter "name"
func getValueOfName(array map[string]alexa.Slot, name string) string {
	var firstName string
	for _, v := range array {
		if v.Name == name {
			firstName = v.Value
		}
	}
	return firstName
}

// Given slots received with the request
// getValueOfNameForUser returns slot value of the slot
// having the struct field "Name" value equal to the string parameter "name"
func getValueOfNameForUser(array []user.Attribute, name string) string {
	var firstName string
	for _, v := range array {
		if v.Name == name {
			firstName = v.Value
		}
	}
	return firstName
}

// fetchNationalityPredictions sends a network request to nationalize api to
// make nationality guesses for a particular first name
func fetchNationalityPredictions(name string) nationality.Response {
	response, err := http.Get(fmt.Sprintf("https://api.nationalize.io?name=%s", name))
	if err != nil {
		fmt.Print(err.Error())
		os.Exit(1)
	}

	responseData, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Fatal(err)
	}

	var predictions nationality.Response
	json.Unmarshal(responseData, &predictions)
	return predictions
}

// fetchCountriesOfCodes takes an array of country
// codes and fetches information about each one of them
func fetchCountriesOfCodes(countryCodes []string) countries.Country {
	response, err := http.Get(fmt.Sprintf("https://restcountries.eu/rest/v2/alpha?codes=%s", strings.Join(countryCodes, ";")))
	if err != nil {
		fmt.Print(err.Error())
		os.Exit(1)
	}

	responseData, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Fatal(err)
	}

	var countries countries.Country
	json.Unmarshal(responseData, &countries)
	return countries
}

// fetchGivenName calls Cognito API with AccessToken provided in
// the request received from alexa to get the
// given (first) name of the user.
func fetchGivenName(accessToken string) string {
	values := map[string]string{"AccessToken": accessToken}
	jsonValue, _ := json.Marshal(values)

	req, err := http.NewRequest("POST", "https://cognito-idp.us-east-2.amazonaws.com/", bytes.NewBuffer(jsonValue))
	if err != nil {
		log.Fatal("Error reading request. ", err)
	}
	req.Header.Set("Content-Type", "application/x-amz-json-1.1")
	req.Header.Set("Content-Length", "1162")
	req.Header.Set("X-Amz-Target", "AWSCognitoIdentityProviderService.GetUser")
	req.Header.Set("Content-Length", "1162")
	client := http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal("Error reading response. ", err)
	}

	responseData, err := ioutil.ReadAll(resp.Body)
	var userData user.User
	json.Unmarshal(responseData, &userData)

	return getValueOfNameForUser(userData.Attributes, "given_name")

}

// Handler is the first function that lambda calls when a request to the skill is made
func Handler(request alexa.Request) (alexa.Response, error) {
	return IntentDispatcher(request), nil
}

// IntentDispatcher specifies which intent was fired, then processes it with appropriate handler
func IntentDispatcher(request alexa.Request) alexa.Response {
	var response alexa.Response
	switch request.Body.Intent.Name {
	case alexa.HelpIntent:
		response = HandleHelpIntent(request)
	case "AboutIntent":
		response = HandleAboutIntent(request)
	case "GuessIntent":
		response = HandleGuessIntent(request, false)
	case "GuessWithAccountIntent":
		response = HandleGuessIntent(request, true)
	default:
		response = HandleAboutIntent(request)
	}
	return response
}

// entrypoint to the app
func main() {
	lambda.Start(Handler)
}
