from locust import HttpUser, task, between, events
import time
import logging
import urllib3
urllib3.disable_warnings(urllib3.exceptions.InsecureRequestWarning)

class ApiUser(HttpUser):
    wait_time = between(1, 5)
    auth_token = None
    
    def on_start(self):
        self.client.verify = False 
        self.email = f"test.user.{time.time_ns()}@example.com"
        self.password = "password123"
        self.username = f"locust_user_{time.time_ns()}"

        reg_response = self.client.post("/api/auth/register", json={
            "email": self.email,
            "password": self.password
        })
        
        if reg_response.status_code != 201 and reg_response.status_code != 409:
            logging.warning(f"Registration failed with status {reg_response.status_code}: {reg_response.text}")
        
        login_response = self.client.post("/api/auth/login", json={
            "email": self.email,
            "password": self.password
        })

        if login_response.status_code == 200:
            self.auth_token = login_response.json().get("access_token")
            self.client.headers["Authorization"] = f"Bearer {self.auth_token}"
            
            # After logging in, create a profile
            profile_response = self.client.post("/api/profile/me", json={
                "username": self.username,
                "bio": "I am a locust user, testing the system."
            })
            if profile_response.status_code != 200:
                logging.warning(f"Failed to create profile for {self.email}. Status: {profile_response.status_code}")

        else:
            logging.error(f"Failed to login user {self.email}, stopping user. Status: {login_response.status_code}, Response: {login_response.text}")
            self.environment.runner.quit()

    @task(10)
    def post_new_status(self):
        self.client.post("/api/posts/me", json={
            "content": f"Locust post at {time.time()}"
        })

    @task(5)
    def get_own_posts(self):
        self.client.get("/api/posts/me")

    @task(3) 
    def get_feed(self):
        self.client.get("/api/feed")

    @task(1)
    def get_own_profile(self):
        self.client.get("/api/profile/me")

    @task(1)
    def get_friends(self):
        self.client.get("/api/friends")