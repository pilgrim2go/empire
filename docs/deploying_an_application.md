# Deploying an Application

Before using Empire successfully it's important to understand its design and architecture. It's suggested you first read the [Features & Design Goals document](http://empire.readthedocs.org/en/latest/features_and_design_goals/).

There are a few key things which your applications, and overall architecture, will need to keep in mind in order to be successful with Empire.  Below is an exhaustive set of criteria your applications must meet in order to work with Empire.  It's not that long and by no means locks you in to Empire.  Structuring your application in this way provides many benefits, only one of which is being able to be managed by Empire.

For the impatient, there are two absolutes:

- You must create a `Procfile` which defines how to run your application
- If your application answers requests over the network, it must listen on port 8080, or better, on the `$PORT` environment variable. It's strongly suggested that you use the `$PORT` variable as port 8080 could change in future releases.

To see an example application, you can look at [remind101/acme-inc]



## Procfile

Empire was modeled after the Heroku API.  As such, many concepts and commands are similar.  One concept which has bled over and which is a requirement is that of a [Procfile][procfile].  For your application to actually work, you must include a Procfile at the root level of your application.

Empire supports two Procfile formats; `standard` and `extended`. The `standard` format is probably what you're used to if you've used Heroku before and simply maps a process name to a command to run:

```yaml
worker: celery -A tasks worker --loglevel=info
```

The `extended` format is Empire specific and allows you to define additional configuration for processes, like exposure settings (http/https/tcp/ssl), health checks, and more. An example of the Procfile above in the `extended` format would simply be:

```yaml
worker:
  command: celery -A tasks worker --loglevel=info
```

The extended Procfile formation is documented [here][extended-procfile].

Whichever format you use, the file would be named `Procfile`, and live at the directory root for your application.

```console
$ tree .
.
├── Dockerfile
├── Procfile
├── myapplication/

```


## Empire application types

Empire treats "web" and "non-web" services different.  Here, a "web" process is defined as anything which needs to expose a port.  The way to differentiate the two types of processes is easy.

### Non-web processes

Let's start with non-web processes first since they're much simpler.  If your service doesn't expose a port it doesn't need to be discovered by any other system.  With that, all you need to do it deploy your application and let Empire manage it. The act of discovering *other* services is up to you (database, caches, etc.)

### Web processes

If your service is going to be used by other systems, you'll need to run some type of server which exposes a port.  In this scenario you'll need to name it `web` in your Procfile:

```
web: node server.js
```

or, using Django as an example:

```
web: python manage.py runserver
```

For web applications, Empire does a few things.  There are multiple layers of routing, none of which are extremely complex.

Assume we have deployed a container named `mycompany/awesome-app`.

The routing to your application is handled as such inside of the VPC:

```

     http://awesome-app/
              +
              |
              |
              v
             ELB: port 80
              +
              |
              |
              v
            Minion: port [9000-10000]
              +
              |
              |
              v
          Container: port $PORT

```

There are various things going on...let's break them down:

- Empire creates an internal HostedZone CNAME record for `awesome-app`.  This CNAME points at an ELB which Empire also creates, specifically for this application.  If you `emp deploy mycompany/another-app`, yet *another* CNAME and ELB would be created for `another-app`.
- The ELB created is managed by Empire. As you scale your application *up*, instances running your application are placed into the ELB.  Likewise, as you scale your application *down*, instances are removed from the ELB. The ELB listens on port 80 and maps to a random port between 9000 and 10000 on the minion instances running your application.
- The ELB runs a health check to determine whether your application is healthy. It will simply perform a tcp `ping` to your application...if your app doesn't respond, you will end up in a state where there are no healthy instances behind the ELB.
- The container running on a minion will map a random port between 9000 and 10000 to the `$PORT` environment variable in your application.  Currently, `$PORT` defaults to 8080. The random port in the 9000-10000 range is managed by Empire.


There are a few other environment variables which Empire will set for your running container, but `$PORT` is the most important for now.  It's **strongly** suggested that you use `$PORT` rather than the default of 8080.  There are various ways to get your application listening on `$PORT`. One such way is to run your application from a shell script, which is in turn used in your `Procfile`

`run.sh`

```
#!/bin/bash

gunicorn -w 2 --bind=:$PORT app:app
```

`Dockerfile`

```
RUN mkdir /code
ADD . /code/
```

`Procfile`

```
web: /code/run.sh
```

### Scheduled processes

**NOTE:** This feature is currently experimental, and requires the CloudFormation backend.

Processes that should run at scheduled times can be configured in the extended procfile by setting a `cron` expression:

```
scheduled-job:
  command: ./bin/scheduled-job
  cron: '0/2 * * * ? *' // Run once every 2 minutes
```

Like other non-web processes, scheduled processes are disabled by default. To enable a scheduled job, simply scale it up:

```console
$ emp scale scheduled-job=1
```

If you want to run more than 1 instance of the process when the cron expression triggers, you can scale it to a value greater than 1:

```console
$ emp scale scheduled-job=5 # Run 5 of these every minute
```

To disable a scheduled job, simply scale it back down to 0:

```console
$ emp scale scheduled-job=0
```

Refer to http://docs.aws.amazon.com/AmazonCloudWatch/latest/DeveloperGuide/ScheduledEvents.html for details on the cron expression syntax.

## Environment variables

TODO

[procfile]: https://devcenter.heroku.com/articles/procfile
[extended-procfile]: https://github.com/remind101/empire/tree/master/procfile
[remind101/acme-inc]: https://github.com/remind101/acme-inc
