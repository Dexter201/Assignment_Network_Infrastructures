
### Project code structure

## Global Project Structure

.
├── INFO8011_Statement.pdf
├── README.md
└── sources
    ├── docker-compose.yaml
    ├── locustfile.py
    ├── prometheus
    │   ├── prometheus.yml
    │   └── rules.yaml
    |
    └── services
        ├── feed-service
        ├── gateway
        ├── load-balancer
        ├── post-service
        └── user-service

# Important Information to start the project
It's important to know that the locustfile used for testing and the docker-compose.yml to launch the containers/services is in the of the Assignment folder. 
We need to be in the root directory to start docker and locust.

## General structure
README in the root
The rest of the code is located in sources

Each service holds it's own dockerfile.
In addition, my 3 services: feed-service, gateway and load-balancer also hold the go.mod and go.sum files that are the dependency files 
copied by the dockerfile --> I chose to do this because it is more efficiant with docker to just copy the dependencies i need into the containers 
and it also helps to make vscode not constantly cry (red errors everywhere) and remove vscode errors. Win -Win

# prometheus
This folder just holds the prometheus configuration files and it's rules --> mounted as volumes in the docker

# services
This folder holds all our 5 services

# post-service and user-services
folder holds the source code of the post and user service, given by the instructions and unchanged

# feed-service

├── feed-service
│   ├── Dockerfile
│   ├── feed.go
│   ├── go.mod
│   ├── go.sum
│   ├── main.go
│   └── utils.go


entrypoint is main.go: starts the http server and intilizes the feed handler

feed.go is the main source code to implement the feed service

utils.go are some utils fucntions for the feed - service like loading environment variables and a healthcheck for debugging


# gateway

├── gateway
│   ├── auth.go
│   ├── cert.pem
│   ├── Dockerfile
│   ├── go.mod
│   ├── go.sum
│   ├── key.pem
│   ├── main.go
│   ├── metrics.go
│   ├── proxy.go
│   ├── router.go
│   └── utils.go

entrypoint is main.go: it starts an https server with the self signed certificates
here the router, authentification middlerware, metrics middleware and the proxy middleware are initilized

utils.go are some utils fucntion for the gateway 

auth.go is the source code that is related to everything that comes with authentification and communication with the database
it implements the authentification middleware

metrics.go is the source code that is related to metrics analyzing and saving
it implements the metrics middleware

proxy.go is the source code related to the proxy middleware --> its job is to forward traffic correctly and set the correct headers

router.go implements our router , it creates the necessary proxies and exposes the endpoints needed to make the project work


# load-balancer

├── load-balancer
│   ├── Dockerfile
│   ├── go.mod
│   ├── go.sum
│   ├── healthcheck.go
│   ├── lb.go
│   ├── main.go
│   ├── rateLimiter.go
│   └── utils.go


entrypoint: main.go creates the loadbalancer based on the environment variables and starts the http server 

lb.go implements the entire laod balancer logic specifically the handling of the 3 algorithms and the forwarding of traffic to the correct backend

rateLimiter.go implements a wraper around a reader to throttle the rate

healthcheck.go is the file that implemnts everything related to healthchecking


### Instructions to run the Project and check the tests (locust and prometheus):

How to Test the Platform with Locust

This guide explains how to set up and run the Locust load test against the running microservice platform.

It assumes your Docker containers are already running 

# Step 1: Create a Python Virtual Environment

Your OS (Linux/macOS) may prevent you from installing packages globally. The best practice is to create a virtual environment for the Python packages.

From the project's root directory:

# Create a virtual environment named .venv
python3 -m venv .venv

# Activate it (this changes your terminal session)
source .venv/bin/activate


You will see your terminal prompt change to show (.venv), indicating the environment is active.

# Step 2: Install Locust

With your virtual environment active, install Locust using pip:

pip install locust


# Step 3: Run the Test

Start the Locust test from the project's root directory:

locust -f locustfile.py --host https://localhost


# Step 4: Run the Swarm from the Web UI

Once you run that command, your terminal will print a message:
Starting web interface at http://localhost:8089

Open.

You'll see the "Start new swarm" page.

Enter the number of users to simulate.

Enter the "Spawn rate" .

In my tests I often set 100 - 200 and 5 - 10 respectively

Click "Start swarm".


# Step 5 Prometheus:
While the test is running, open http://localhost:9090 (the Prometheus dashboard).

3 rules can be found in Status:Rule health 
you can copy them and inject them in the query section of the homepage and then check the graphs

# Step 6: Stop the Test

When you are finished:

Click "Stop" in the Locust web UI.

Go back to your terminal and press Ctrl+C to stop Locust.

Type deactivate to exit the Python virtual environment.

Run docker compose down -v to stop and remove all containers, networks, and database volumes.