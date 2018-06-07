const strava = require('strava-v3');
const express = require('express');
const fs = require('fs');
const path = require('path');
const opn = require('opn');

const app = express();

var request = require('request-promise');
var request = request.defaults({jar: true, simple: false});

process.env.STRAVA_ACCESS_TOKEN = "empty";
process.env.STRAVA_CLIENT_ID = "24949";
process.env.STRAVA_CLIENT_SECRET = "efa786ccc73e8d9d61f8a180f4cba2fe1a430b83";
process.env.STRAVA_REDIRECT_URI = "http://localhost:3000/exchange_token";

jsonFormater = (json) => {
  return JSON.stringify(creds).replace(/,/g, ',\n  ')
                              .replace(/\{/g, '{\n  ')
                              .replace(/\}/g, '\n}')
                              .replace(/\:/g, ': ');
}


// read creds from file
var jsonPath = path.join(__dirname, '/credentials.json');
try {
  var creds = fs.readFileSync(jsonPath).toString();
} catch (err) {
  console.error('ERROR: Could not find credentials.json');
  process.exit(1);
}

// parse file content
try {
  creds = JSON.parse(creds);
} catch (err) {
  console.error('ERROR: Could not parse credentials.json');
  process.exit(1);
}

noGui = () => {
  request.post({ url: 'https://we-row.mynohrd.com/login', form: {email: creds.WEROW_EMAIL, password: creds.WEROW_PASSWORD} })
         .then(function (body) {
            request.get('https://we-row.mynohrd.com/history/races')
                   .then(function (body) {

                      try {
                        races = JSON.parse(body);
                      } catch (err) {
                        console.error('ERROR: we-row credentials are most likely wrong');
                        process.exit(1);
                      }

                      handleRaceIndex(races);
                    })
                   .catch(function (err) {
                      console.error('ERROR: Problem getting we-row races');
                      process.exit(1);
                   });
         })
         .catch(function (err) {
            console.error('ERROR: Problem logging into we-row');
            process.exit(1);
         });
}

if (creds.STRAVA_ACCESS_TOKEN == "") {
  app.listen(3000)
  // no strava access token, start oauth flow
  opn('http://localhost:3000/');
  console.log('INFO: If your browser didnt open, please visit http://localhost:3000')
} else if (creds.WEROW_EMAIL == "" || creds.WEROW_PASSWORD == "") {
  app.listen(3000)
  // start we-row auth process
  opn('http://localhost:3000/exchange_token');
  console.log('INFO: If your browser didnt open, please visit http://localhost:3000/exchange_token')
} else {
  noGui();
}


app.get('/', function (req, res) {
  if (creds.STRAVA_ACCESS_TOKEN == "") {
    // oauth flow
    const oauthUrl = strava.oauth.getRequestAccessURL({scope:"view_private,write"})
    res.redirect(307, oauthUrl);
  } else {
    // start we-row auth process
    res.redirect(307, '/exchange_token');
  }
})

app.get('/exchange_token', function (req, res) {
  if (typeof req.query.code !== 'undefined') {
    // we came here from strava
    strava.oauth.getToken(req.query.code,function(err,payload,limits) {
        if(!err) {
          creds.STRAVA_ACCESS_TOKEN = payload.access_token;
          process.env.STRAVA_ACCESS_TOKEN = payload.access_token;
          fs.writeFileSync(jsonPath, jsonFormater(creds), function(err) {
              if(err) {
                  console.error('ERROR: Could not write credentials.json');
                  process.exit(1);
              }
          });
        } else {
          console.error('ERROR: Problem with OAuth flow (see messagein browser)');
          res.send(err);
        }
    });
  }

  if (creds.WEROW_EMAIL == "" || creds.WEROW_PASSWORD == "") {
    // we-row auth process
    res.send(`
      <html><body>
        <p>All good. Now please enter your We-Row account details so we can fetch the data.</p>
        <form action="/werow" method="get">
          <input placeholder="Your e-mail address" value="" name="email" type="email" id="email" autocomplete="on">
          <input placeholder="Your password" name="password" type="password" value="" id="password" autocomplete="off">
          <br/><button type="submit">Submit</button>
        </form>
      </html></body>
    `);
  } else {
    res.redirect(307, '/werow?email=' + creds.WEROW_EMAIL + '&password=' + creds.WEROW_PASSWORD);
  }
})

app.get('/werow', function (req, res) {
  let email = req.query.email;
  let password = req.query.password;

  request.post({ url: 'https://we-row.mynohrd.com/login', form: {email: email, password: password} })
         .then(function (body) {
            request.get('https://we-row.mynohrd.com/history/races')
                   .then(function (body) {

                      try {
                        races = JSON.parse(body);
                      } catch (err) {
                        console.error('ERROR: we-row credentials are most likely wrong');
                        process.exit(1);
                      }

                      creds.WEROW_EMAIL = email;
                      creds.WEROW_PASSWORD = password;
                      fs.writeFileSync(jsonPath, jsonFormater(creds), function(err) {
                          if(err) {
                              console.error('ERROR: Could not write credentials.json');
                              process.exit(1);
                          }
                      });

                      handleRaceIndex(races);
                      res.send(`
                        <html><body>
                          <p>Working ...</p>
                          <p>You can close this window now and should move your attention to the command line.</p>
                        </html></body>
                      `);
                    })
                   .catch(function (err) {
                      console.error('ERROR: Problem getting we-row races');
                      process.exit(1);
                   });
         })
         .catch(function (err) {
            console.error('ERROR: Problem logging into we-row');
            process.exit(1);
         });
})

handleRaceIndex = async (races) => {

  var raceArray = await Promise.all(races.map(async (race) => {
    if (race.state == "finished") {
      return await request.get("https://we-row.mynohrd.com/history/races/data/" + race.id)
    }
  }));

  opn('https://www.strava.com/athlete/training_activities?activity_type=Rowing');
  console.log('INFO: If your browser didnt open, please visit https://www.strava.com/athlete/training_activities?activity_type=Rowing')
  console.log(raceArray.filter(race => race).length + ' Activities found on we-row');

  // for (const [index, el] of raceArray.entries()) {
  raceArray.forEach(function(el, index) {

    try {
      var race = JSON.parse(el);
    } catch {
      return;
    }

    var date = new Date(race.started_at);
    var offset = date.getTimezoneOffset()/60;
    var start_date_local = date.setHours(date.getHours()-offset);
    var start_date_local = date.toISOString();
    var elapsed_time = Math.floor(race.time/1000);

    var dataPoints = 0;
    var speed = 0;
    var strokeRate = 0;

    for (var j in race.data.raceData) {
      dataPoints++;
      speed = speed + race.data.raceData[j].speed;
      strokeRate = strokeRate + race.data.raceData[j].stroke_rate;
    }

    var speedAvg = Number(Math.round(speed / dataPoints+'e2')+'e-2');
    var strokeRateAvg = Number(Math.round(strokeRate / dataPoints+'e2')+'e-2');

    var desc = 'Average Speed: ' + speedAvg + " m/s\nAverage Stroke Rate: " + strokeRateAvg;

    var args = {
      'access_token': creds.STRAVA_ACCESS_TOKEN,
      'before': parseInt(String(race.started_at).substring(0,10))+1,
      'after': parseInt(String(race.started_at).substring(0,10))-1,
    };

    // Does acticity already exist?
    strava.athlete.listActivities(args, function(err,payload,limits) {
      if(!err) {
        // is new activity
        if (payload.length == 0) {
          console.log('session not found on strava, uploading...');
          createArgs = {
            'access_token': creds.STRAVA_ACCESS_TOKEN,
            'name': 'Rowing Session',
            'type': 'Rowing',
            'start_date_local': start_date_local,
            'elapsed_time': elapsed_time,
            'description': desc,
            'distance': race.distance,
            'private': true,
          };

          strava.activities.create(createArgs, function(err,payload,limits) {
            if(!err) {
              updateArgs = {
                'access_token': creds.STRAVA_ACCESS_TOKEN,
                'id': payload.id,
                'trainer': 1,
              };

              strava.activities.update(updateArgs, function(err,payload,limits) {
                if(err) {
                    console.error('ERROR: Problem while updating activity.');
                }
              });
            } else {
              console.error('ERROR: Problem while creating activity.');
            }
          });
        } else {
          console.log('session already uploaded, skipping...');
        }
      } else {
        console.error('ERROR: Problem while reading activities.');
      }
    });
  });
}