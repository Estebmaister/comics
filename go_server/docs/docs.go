// Package docs Code generated by swaggo/swag. DO NOT EDIT
package docs

import "github.com/swaggo/swag"

const docTemplate = `{
    "schemes": {{ marshal .Schemes }},
    "swagger": "2.0",
    "info": {
        "description": "{{escape .Description}}",
        "title": "{{.Title}}",
        "termsOfService": "http://swagger.io/terms/",
        "contact": {
            "name": "Estebmaister",
            "url": "http://www.github.com/estebmaister",
            "email": "estebmaister@gmail.com"
        },
        "license": {
            "name": "Apache 2.0",
            "url": "http://www.apache.org/licenses/LICENSE-2.0.html"
        },
        "version": "{{.Version}}"
    },
    "host": "{{.Host}}",
    "basePath": "{{.BasePath}}",
    "paths": {
        "/admin/dashboard": {
            "get": {
                "security": [
                    {
                        "Bearer JWT": []
                    }
                ],
                "description": "Returns the admin dashboard, needs admin auth",
                "consumes": [
                    "application/json"
                ],
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "Dashboard"
                ],
                "summary": "Dashboard",
                "operationId": "dashboard",
                "parameters": [
                    {
                        "type": "string",
                        "default": "Bearer XXX",
                        "description": "Bearer JWT",
                        "name": "Authorization",
                        "in": "header",
                        "required": true
                    }
                ],
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "type": "object",
                            "additionalProperties": {
                                "type": "string"
                            }
                        }
                    },
                    "400": {
                        "description": "Not registered",
                        "schema": {
                            "type": "string"
                        }
                    },
                    "404": {
                        "description": "Not implemented",
                        "schema": {
                            "type": "string"
                        }
                    }
                }
            }
        },
        "/login": {
            "post": {
                "security": [
                    {
                        "Bearer JWT": []
                    }
                ],
                "description": "Login a user with basic credentials to receive an auth 'token'\nin the headers if successful",
                "consumes": [
                    "application/json"
                ],
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "Authentication"
                ],
                "summary": "Login existent user",
                "operationId": "user-login",
                "parameters": [
                    {
                        "type": "string",
                        "default": "Bearer XXX",
                        "description": "Bearer JWT",
                        "name": "Authorization",
                        "in": "header"
                    },
                    {
                        "description": "Login user",
                        "name": "user",
                        "in": "body",
                        "required": true,
                        "schema": {
                            "$ref": "#/definitions/domain.LoginRequest"
                        }
                    }
                ],
                "responses": {
                    "200": {
                        "description": "logged in",
                        "schema": {
                            "$ref": "#/definitions/domain.AuthResponse"
                        },
                        "headers": {
                            "Authorization": {
                                "type": "string",
                                "description": "Bearer JWT"
                            }
                        }
                    },
                    "400": {
                        "description": "no ok",
                        "schema": {
                            "type": "string"
                        }
                    }
                }
            }
        },
        "/protected/profile": {
            "get": {
                "security": [
                    {
                        "Bearer JWT": []
                    }
                ],
                "description": "Function for getting the user profile",
                "consumes": [
                    "application/json"
                ],
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "Profile"
                ],
                "summary": "Profile",
                "operationId": "profile",
                "parameters": [
                    {
                        "type": "string",
                        "default": "Bearer XXX",
                        "description": "Bearer JWT",
                        "name": "Authorization",
                        "in": "header",
                        "required": true
                    }
                ],
                "responses": {
                    "200": {
                        "description": "ok",
                        "schema": {
                            "type": "object",
                            "additionalProperties": {
                                "type": "string"
                            }
                        }
                    },
                    "400": {
                        "description": "not registered",
                        "schema": {
                            "type": "integer"
                        }
                    },
                    "404": {
                        "description": "not registered",
                        "schema": {
                            "type": "string"
                        }
                    }
                }
            }
        },
        "/refresh-token": {
            "post": {
                "description": "Function for refreshing the access token",
                "consumes": [
                    "application/json"
                ],
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "Authentication"
                ],
                "summary": "RefreshToken",
                "operationId": "refresh-token",
                "parameters": [
                    {
                        "type": "string",
                        "default": "Bearer refresh_token",
                        "description": "Bearer JWT",
                        "name": "Authorization",
                        "in": "header",
                        "required": true
                    },
                    {
                        "enum": [
                            "user",
                            "admin"
                        ],
                        "type": "string",
                        "description": "role",
                        "name": "Role",
                        "in": "header"
                    }
                ],
                "responses": {
                    "200": {
                        "description": "new access token generated",
                        "schema": {
                            "$ref": "#/definitions/domain.AuthResponse"
                        },
                        "headers": {
                            "Authorization": {
                                "type": "string",
                                "description": "Bearer JWT"
                            }
                        }
                    },
                    "400": {
                        "description": "not registered",
                        "schema": {
                            "type": "integer"
                        }
                    },
                    "404": {
                        "description": "not registered",
                        "schema": {
                            "type": "string"
                        }
                    }
                }
            }
        },
        "/signup": {
            "post": {
                "description": "Signs Up a new user (for demonstration purposes),\nreceive a confirmation for success or failure",
                "consumes": [
                    "application/json"
                ],
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "Authentication"
                ],
                "summary": "SignUp new user",
                "operationId": "user-signup",
                "parameters": [
                    {
                        "description": "Login user",
                        "name": "user",
                        "in": "body",
                        "required": true,
                        "schema": {
                            "$ref": "#/definitions/domain.SignUpRequest"
                        }
                    }
                ],
                "responses": {
                    "201": {
                        "description": "registered",
                        "schema": {
                            "$ref": "#/definitions/domain.AuthResponse"
                        }
                    },
                    "400": {
                        "description": "not registered, invalid data",
                        "schema": {
                            "$ref": "#/definitions/domain.NoDataResponse"
                        }
                    },
                    "409": {
                        "description": "username or email already in use",
                        "schema": {
                            "$ref": "#/definitions/domain.NoDataResponse"
                        }
                    }
                }
            }
        }
    },
    "definitions": {
        "domain.AuthData": {
            "type": "object",
            "properties": {
                "access_token": {
                    "type": "string"
                },
                "refresh_token": {
                    "type": "string"
                },
                "user_id": {
                    "type": "string"
                }
            }
        },
        "domain.AuthResponse": {
            "type": "object",
            "properties": {
                "data": {
                    "$ref": "#/definitions/domain.AuthData"
                },
                "message": {
                    "type": "string"
                },
                "status": {
                    "type": "integer"
                }
            }
        },
        "domain.LoginRequest": {
            "type": "object",
            "required": [
                "email",
                "password"
            ],
            "properties": {
                "email": {
                    "type": "string",
                    "example": "test@example.com"
                },
                "password": {
                    "type": "string",
                    "example": "password123"
                }
            }
        },
        "domain.NoDataResponse": {
            "type": "object",
            "properties": {
                "data": {
                    "type": "boolean"
                },
                "message": {
                    "type": "string"
                },
                "status": {
                    "type": "integer"
                }
            }
        },
        "domain.SignUpRequest": {
            "type": "object",
            "required": [
                "email",
                "password",
                "username"
            ],
            "properties": {
                "email": {
                    "type": "string",
                    "example": "test@example.com"
                },
                "password": {
                    "type": "string",
                    "example": "password123"
                },
                "username": {
                    "type": "string",
                    "example": "testuser"
                }
            }
        }
    },
    "securityDefinitions": {
        "Bearer JWT": {
            "description": "Type \"Bearer\" followed by a space paste the JWT.",
            "type": "apiKey",
            "name": "Authorization",
            "in": "header"
        }
    }
}`

// SwaggerInfo holds exported Swagger Info so clients can modify it
var SwaggerInfo = &swag.Spec{
	Version:          "1.1",
	Host:             "",
	BasePath:         "",
	Schemes:          []string{},
	Title:            "Comics API",
	Description:      "Server documentation to query comics from the DB.",
	InfoInstanceName: "swagger",
	SwaggerTemplate:  docTemplate,
	LeftDelim:        "{{",
	RightDelim:       "}}",
}

func init() {
	swag.Register(SwaggerInfo.InstanceName(), SwaggerInfo)
}
