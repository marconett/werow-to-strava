# werow-to-strava

This is spaghetti code, but it works pretty well. I wasn't able to detect when it's done importing all activities to Strava (I still don't really get async/await), so the program needs to be killed with ctrl+c when it's done.

It goes through all We-Row sessions every time it's run and imports those to Strava. It detects already imported rowing sessions (and skips them) by comparing the start time of the Strava activity to the start time of the We-Row session. There might be some daylight saving related problems with this method, but I'm not sure how those cases are handles by Strava and We-Row.

Note: After the first successful run, it saves your We-Row credentials and Strava Access Token as plain text in `credentials.json`.

## Usage
* Clone
* `yarn install`
* `node .`
* Follow the Strava OAuth flow and enter We-Row credentials in your browser.
* On the subsequent runs, no browser interaction is needed.

## Go version

I tried implementing this in go, but ran into problems with the official Strava Golang library (https://github.com/strava/go.strava/) being out of date.