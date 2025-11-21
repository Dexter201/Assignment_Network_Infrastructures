from locust import HttpUser, task, between, events
import time
import logging
import random
import urllib3

# Disable TLS warnings for self-signed certs
urllib3.disable_warnings(urllib3.exceptions.InsecureRequestWarning)

class ApiUser(HttpUser):
    wait_time = between(1, 5)

    # This list is shared across all instances of ApiUser.
    # we will append the real UUIDs here for the add and delete friend tasks
    registered_users = []
    
    auth_token = None
    my_uuid = None
    friend_ids = []

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
            logging.warning(f"Registration failed: {reg_response.status_code}")
        
        login_response = self.client.post("/api/auth/login", json={
            "email": self.email,
            "password": self.password
        })

        if login_response.status_code == 200:
            self.auth_token = login_response.json().get("access_token")
            self.client.headers["Authorization"] = f"Bearer {self.auth_token}"
            
            # Create Profile (POST /api/profile/me) 
            profile_response = self.client.post("/api/profile/me", json={
                "username": self.username,
                "bio": "I am a locust user."
            })
            
            # Fetch Own Profile to get UUID (GET /api/profile/me) and append to register list
            # We need 'my_uuid' to test the specific {userId} endpoints
            if profile_response.status_code == 200:
                 
                 get_profile = self.client.get("/api/profile/me")
                 if get_profile.status_code == 200:
                     self.my_uuid = get_profile.json().get("uuid")
                     if self.my_uuid:
                         ApiUser.registered_users.append(self.my_uuid)
        else:
            logging.error(f"Login failed for {self.email}. Stopping.")
            self.environment.runner.quit()

    # --- POST SERVICE ENDPOINTS ---

    # POST /api/posts/me 
    @task(10)
    def post_new_status(self):
        self.client.post("/api/posts/me", json={
            "content": f"Locust post at {time.time()}"
        })

    #GET /api/posts/me 
    @task(5)
    def get_own_posts(self):
        self.client.get("/api/posts/me")

    #GET /api/posts/{userId}
    @task(3)
    def get_specific_user_posts(self):

        # We test this using our own uuid if we have no friends yet
        target_id = self.my_uuid
        if self.friend_ids:
            target_id = random.choice(self.friend_ids)
            
        if target_id:
            self.client.get(f"/api/posts/{target_id}", name="/api/posts/[userId]")

    # --- FEED SERVICE ENDPOINTS ---

    #GET /api/feed 
    @task(5) 
    def get_feed(self):
        self.client.get("/api/feed")

    # --- USER SERVICE ENDPOINTS ---

    #GET /api/profile/me
    @task(1)
    def get_own_profile(self):
        self.client.get("/api/profile/me")

    # GET /api/profile/{userId} 
    @task(2)
    def get_specific_user_profile(self):
        target_id = self.my_uuid
        if self.friend_ids:
            target_id = random.choice(self.friend_ids)

        if target_id:
            self.client.get(f"/api/profile/{target_id}", name="/api/profile/[userId]")

    # GET /api/friends
    @task(2)
    def get_friends(self):

        with self.client.get("/api/friends", catch_response=True) as response:
            if response.status_code == 200:
                try:
                    data = response.json()
                    if data is None:
                        self.friend_ids = []
                    else:
                        self.friend_ids = data
                except:
                    self.friend_ids = [] # Handle cases where response is empty or invalid JSON

    # POST /api/friends 
    @task(2)
    def add_friend(self):
        if self.friend_ids is None:
            self.friend_ids = []

        # at least 2 users in the system to make a friendship
        if len(ApiUser.registered_users) < 2:
            return

        # find all users in the system who are not me
        potential_friends = [u for u in ApiUser.registered_users if u != self.my_uuid]
        
        #exclude people that are already friends 
        potential_friends = [u for u in potential_friends if u not in self.friend_ids]

        if not potential_friends:
            return

        target_uuid = random.choice(potential_friends)

        self.client.post("/api/friends", json={
            "friend_uuid": target_uuid
        }, name="/api/friends (add)")

    # DELETE /api/friends 
    @task(1)
    def remove_friend(self):
        if self.friend_ids:
            target_to_remove = random.choice(self.friend_ids)
            self.client.delete("/api/friends", json={
                "friend_uuid": target_to_remove
            }, name="/api/friends (delete)")