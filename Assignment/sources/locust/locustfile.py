from locust import HttpUser, task, between, events
import time
import logging # Import logging to print messages

class ApiUser(HttpUser):
    """
    Simulates a full user flow:
    1. Register / Login on start
    2. Use the auth token to browse API endpoints
    """
    
    # Users will wait between 1 and 5 seconds before starting a new task
    wait_time = between(1, 5)
    auth_token = None
    
    def on_start(self):
        """
        Called when a virtual user is started.
        Simulates registration (first time) or login.
        """
        self.email = f"test.user.{time.time_ns()}@example.com"
        self.password = "password123"

        reg_response = self.client.post("/api/auth/register", json={
            "email": self.email,
            "password": self.password
        })
        

        if reg_response.status_code != 201 and reg_response.status_code != 409:
            # 201 = Created, 409 = Conflict (user already exists)
            logging.warning(f"Registration failed with status {reg_response.status_code}: {reg_response.text}")
        
        login_response = self.client.post("/api/auth/login", json={
            "email": self.email,
            "password": self.password
        })

        if login_response.status_code == 200:
            self.auth_token = login_response.json().get("access_token")
            self.client.headers["Authorization"] = f"Bearer {self.auth_token}"
        else:
            # If login fails, stop this user.
            logging.error(f"Failed to login user {self.email}, stopping user. Status: {login_response.status_code}, Response: {login_response.text}")
            self.environment.runner.quit()

    @task(3) 
    def get_feed(self):
        """
        Task to fetch the main feed.
        This will hit /api/feed and REQUIRES auth.
        """
        self.client.get("/api/feed")

    @task(1)
    def get_own_profile(self):
        """ 
        Task to fetch the user's own profile. 
        """
        self.client.get("/api/profile/me")

    @task(1)
    def get_friends(self):
        """ 
        Task to fetch the user's friends list. 
        """
        self.client.get("/api/friends")