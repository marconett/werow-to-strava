// oauth_example.go provides a simple example implementing Strava OAuth
// using the go.strava library.
//
// usage:
//   > go get github.com/strava/go.strava
//   > cd $GOPATH/github.com/strava/go.strava/examples
//   > go run oauth_example.go -id=youappsid -secret=yourappsecret
//
//   Visit http://localhost:8080 in your webbrowser
//
//   Application id and secret can be found at https://www.strava.com/settings/api
package main

import (
  "flag"
  "fmt"
  "net/http"
  "github.com/strava/go.strava"
  "net/url"
  "io/ioutil"
  // "io"
  // "bytes"
  "sync"
  "os"
  "encoding/json"
  "strconv"
  "errors"
  "time"
)

const port = 8080 // port of local demo server

var authenticator *strava.OAuthAuthenticator

type Jar struct {
    lk      sync.Mutex
    cookies map[string][]*http.Cookie
}

func NewJar() *Jar {
    jar := new(Jar)
    jar.cookies = make(map[string][]*http.Cookie)
    return jar
}

// SetCookies handles the receipt of the cookies in a reply for the
// given URL.  It may or may not choose to save the cookies, depending
// on the jar's policy and implementation.
func (jar *Jar) SetCookies(u *url.URL, cookies []*http.Cookie) {
    jar.lk.Lock()
    jar.cookies[u.Host] = cookies
    jar.lk.Unlock()
}

// Cookies returns the cookies to send in a request for the given URL.
// It is up to the implementation to honor the standard cookie use
// restrictions such as in RFC 6265.
func (jar *Jar) Cookies(u *url.URL) []*http.Cookie {
    return jar.cookies[u.Host]
}

func FloatToString(number float64) string {
    return strconv.FormatFloat(number, 'f', 0, 64)
}

func IntToString(number int64) string {
    return strconv.FormatInt(number, 10)
}

func responseHandler(resp *http.Response, err error) (string, error) {
    resource := resp.Request.URL.String()

    if (err == nil) && (resp.StatusCode == 200) {

      body, err := ioutil.ReadAll(resp.Body)
      resp.Body.Close()

      if (err != nil) {
        return "", errors.New(resource + ": Error parsing login response");
      }

      if (resource == "https://we-row.mynohrd.com/") {
        return "", errors.New(resource + ": Wrong username and password OR unauthorized");
      }

      return string(body), nil;
    } else {
        return "", errors.New(resource + ": Error with login request");
    }

    return "", errors.New("Error: Dunno");
}

func main() {
  // setup the credentials for your app
  // can be found at https://www.strava.com/settings/api
  flag.IntVar(&strava.ClientId, "id", 24949, "Strava Client ID")
  flag.StringVar(&strava.ClientSecret, "secret", "efa786ccc73e8d9d61f8a180f4cba2fe1a430b83", "Strava Client Secret")

  flag.Parse()

  // define a strava.OAuthAuthenticator to hold state.
  // The callback url is used to generate an AuthorizationURL.
  // The RequestClientGenerator can be used to generate an http.RequestClient.
  // This is usually when running on the Google App Engine platform.
  authenticator = &strava.OAuthAuthenticator{
    CallbackURL:            fmt.Sprintf("http://localhost:%d/exchange_token", port),
    RequestClientGenerator: nil,
  }

  http.HandleFunc("/", indexHandler)
  http.HandleFunc("/werow", werowHandler)

  path, err := authenticator.CallbackPath()
  if (err != nil) {
    // possibly that the callback url set above is invalid
    fmt.Println(err)
    os.Exit(1)
  }
  http.HandleFunc(path, authenticator.HandlerFunc(oAuthSuccess, oAuthFailure))

  // start the server
  fmt.Printf("Visit http://localhost:%d/ to view the demo\n", port)
  fmt.Printf("ctrl-c to exit")
  http.ListenAndServe(fmt.Sprintf(":%d", port), nil)

}

func indexHandler(w http.ResponseWriter, r *http.Request) {
  http.Redirect(w, r, authenticator.AuthorizationURL("state1", strava.Permissions.Write, true), 301)
}

func werowHandler(w http.ResponseWriter, r *http.Request) {
  r.ParseForm()
  fmt.Fprint(w, `<p>` + r.Form["email"][0] + `</p>`)
  fmt.Fprint(w, `<p>` + r.Form["stravaToken"][0] + `</p>`)
  fmt.Fprint(w, `<p>` + r.Form["stravaAthlete"][0] + `</p>`)

  stravaClient := strava.NewClient(r.Form["stravaToken"][0])

  jar := NewJar()
  client := &http.Client{
    CheckRedirect: nil,
    Jar: jar,
  }

  // we-row auth
  resp, err := client.PostForm("https://we-row.mynohrd.com/login", url.Values{
      "email": {r.Form["email"][0]},
      "password": {r.Form["password"][0]},
  })
  _, err = responseHandler(resp, err);
  if (err != nil) {
    fmt.Fprint(w, `<p>` + err.Error() + `</p>`)
    // http.Redirect(w, r, "http://localhost:" + IntToString(port) + "/", 301)
  }

  // we-row GET races index
  resp, err = client.Get("https://we-row.mynohrd.com/history/races")
  races, err := responseHandler(resp, err)
  if (err != nil) {
    fmt.Fprint(w, `<p>` + err.Error() + `</p>`)
    // http.Redirect(w, r, "http://localhost:" + IntToString(port) + "/", 301)
  }

  // var racesArray []map[string]interface{}
  var racesJson []map[string]interface{}
  json.Unmarshal([]byte(races), &racesJson)

  for _, v := range racesJson {
    // fmt.Println(v["id"].(float64))

    // we-row GET single race
    resp, err = client.Get("https://we-row.mynohrd.com/history/races/data/" + FloatToString(v["id"].(float64)))
    race, err := responseHandler(resp, err)
    if (err != nil) {
      fmt.Fprint(w, `<p>` + err.Error() + `</p>`)
      // http.Redirect(w, r, "http://localhost:" + IntToString(port) + "/", 301)
    }

    var raceJson map[string]interface{}
    json.Unmarshal([]byte(race), &raceJson)

    // racesArray = append(racesArray, raceJson)

    elapseTime := int(raceJson["time"].(float64)/1000)
    distance := raceJson["distance"].(float64)

    service := strava.NewActivitiesService(stravaClient)
    activity, err := service.Create("Test", strava.ActivityTypes.Rowing, time.Now(), elapseTime).
    Description("test123").
    Distance(distance).
    Do()

    fmt.Println(activity)

    if (err != nil) {
      fmt.Println(err)
    }

    break;


    // fmt.Println(raceJson)

    // TODO: POST stuff somewhere
    // body := new(bytes.Buffer)
    // json.NewEncoder(body).Encode(raceJson)
    // resp, _ := http.Post("https://httpbin.org/post", "application/json; charset=utf-8", body)
    // io.Copy(os.Stdout, resp.Body)
  }
}

func oAuthSuccess(auth *strava.AuthorizationResponse, w http.ResponseWriter, r *http.Request) {
  athlete, _ := json.MarshalIndent(auth.Athlete, "", " ")

  fmt.Fprint(w, `<html><body>`)
  fmt.Fprint(w, `<p>All good. Now please enter your We-Row account details so we can fetch the data.</p>`)
  fmt.Fprint(w, `<form action="/werow" method="post">`)
  fmt.Fprint(w, `<input placeholder="Your e-mail address" value="" name="email" type="email" id="email" autocomplete="on">`);
  fmt.Fprint(w, `<input placeholder="Your password" name="password" type="password" value="" id="password" autocomplete="off">`);
  fmt.Fprint(w, "<input type='hidden' name='stravaToken' value='" + auth.AccessToken + "'>");
  fmt.Fprint(w, "<input type='hidden' name='stravaAthlete' value='" + string(athlete) + "'>");
  fmt.Fprint(w, `<br/><button type="submit">Submit</button>`);
  fmt.Fprint(w, `</form>`)
  fmt.Fprint(w, `</html></body>`)
}

func oAuthFailure(err error, w http.ResponseWriter, r *http.Request) {
  fmt.Fprintf(w, "Authorization Failure:\n")

  // some standard error checking
  if err == strava.OAuthAuthorizationDeniedErr {
    fmt.Fprint(w, "The user clicked the 'Do not Authorize' button on the previous page.\n")
    fmt.Fprint(w, "This is the main error your application should handle.")
  } else if err == strava.OAuthInvalidCredentialsErr {
    fmt.Fprint(w, "You provided an incorrect client_id or client_secret.\nDid you remember to set them at the begininng of this file?")
  } else if err == strava.OAuthInvalidCodeErr {
    fmt.Fprint(w, "The temporary token was not recognized, this shouldn't happen normally")
  } else if err == strava.OAuthServerErr {
    fmt.Fprint(w, "There was some sort of server error, try again to see if the problem continues")
  } else {
    fmt.Fprint(w, err)
  }
}
