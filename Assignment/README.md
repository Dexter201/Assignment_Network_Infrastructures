Instructions for locust use:

How to Test the Platform with Locust

This guide explains how to set up and run the Locust load test against the running microservice platform.

It assumes your Docker containers are already running (e.g., via docker compose up -d).

Step 1: Create a Python Virtual Environment

Your OS (Linux/macOS) may prevent you from installing packages globally. The best practice is to create a "virtual environment" (a self-contained directory) for your Python packages.

From your project's root directory (where your docker-compose.yaml is):

# Create a virtual environment named .venv
python3 -m venv .venv

# Activate it (this changes your terminal session)
source .venv/bin/activate


You will see your terminal prompt change to show (.venv), indicating the environment is active.

Step 2: Install Locust

With your virtual environment active, install Locust using pip:

pip install locust


Step 3: Run the Test

Start the Locust test from your project's root directory (assuming locustfile.py is there):

locust -f locustfile.py --host https://localhost


--host https://localhost: This tells Locust to send all its requests to your gateway, which is listening on port 443 (the default for https://).

Step 4: Run the Swarm from the Web UI

Once you run that command, your terminal will print a message like:
Starting web interface at http://localhost:8089

Open http://localhost:8089 in your browser.

You'll see the "Start new swarm" page.

Enter the number of users to simulate (e.g., 100).

Enter the "Spawn rate" (e.g., 10 users per second).

Click "Start swarm".

Step 5: Watch the Test!

You now have three places to watch the results:

The Locust UI (Browser):

Statistics: Watch the Requests/s climb. Most importantly, check the Failure % column. If this stays at 0%, your API is working correctly!

Charts: See the response times and number of users in real-time.

Failures: If anything goes wrong, errors will appear here.

Your Terminal (Docker Logs):
This is a great way to see all the services working together. In a new terminal, run:

docker compose logs -f


You will see a flood of logs from all your containers:

gateway-1 will show the Metrics: ... logs for every request.

user-load-balancer-1 will show Forwarding connection...

post-load-balancer-1 will show Forwarding connection...

feed-service-1 will show its logs.

Prometheus (Browser):
While the test is running, open http://localhost:9090 (your Prometheus dashboard).

Click on Status > Targets. You should see your gateway job with a green "UP" state.

Click the Graph tab. In the expression bar, type gateway_requests_total and click "Execute". You'll see your metrics being scraped!

Try one of your rules: gateway:requests:total:per_method

Step 6: Stop the Test

When you are finished:

Click "Stop" in the Locust web UI.

Go back to your terminal and press Ctrl+C to stop Locust.

Type deactivate to exit the Python virtual environment.

Run docker compose down -v to stop and remove all containers, networks, and database volumes.