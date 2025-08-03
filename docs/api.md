# API Documentation

## Base URL
`http://localhost:8080`

## Endpoints

### Health Check
- **GET** `/health`
- **Description**: Check if the application is running
- **Response**: 
  ```json
  {
    "status": "healthy",
    "architecture": "gateway"
  }
  ```

### Authentication Module

#### Auth Health Check
- **GET** `/api/auth/health`
- **Description**: Check auth module health
- **Response**:
  ```json
  {
    "status": "healthy",
    "module": "auth"
  }
  ```

#### Auth Status
- **GET** `/api/auth/status`
- **Description**: Get auth module status
- **Response**:
  ```json
  {
    "module": "auth",
    "status": "running",
    "version": "1.0.0"
  }
  ```

#### Login (Not Implemented)
- **POST** `/api/auth/login`
- **Description**: User login
- **Response**:
  ```json
  {
    "message": "Auth module - login endpoint",
    "status": "not_implemented"
  }
  ```

#### Register (Not Implemented)
- **POST** `/api/auth/register`
- **Description**: User registration
- **Response**:
  ```json
  {
    "message": "Auth module - register endpoint",
    "status": "not_implemented"
  }
  ```

### Users Module

#### Users Health Check
- **GET** `/api/users/health`
- **Description**: Check users module health

#### List Users (Not Implemented)
- **GET** `/api/users/`
- **Description**: Get all users

#### Get User (Not Implemented)
- **GET** `/api/users/{id}`
- **Description**: Get user by ID

#### Create User (Not Implemented)
- **POST** `/api/users/`
- **Description**: Create new user

#### Update User (Not Implemented)
- **PUT** `/api/users/{id}`
- **Description**: Update user by ID

#### Delete User (Not Implemented)
- **DELETE** `/api/users/{id}`
- **Description**: Delete user by ID

### Notifications Module

#### Notifications Health Check
- **GET** `/api/notifications/health`
- **Description**: Check notifications module health

#### Get Notifications (Not Implemented)
- **GET** `/api/notifications/`
- **Description**: Get all notifications

#### Send Notification (Not Implemented)
- **POST** `/api/notifications/`
- **Description**: Send a notification

#### Mark as Read (Not Implemented)
- **PUT** `/api/notifications/{id}`
- **Description**: Mark notification as read