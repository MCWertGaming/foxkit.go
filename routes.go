package foxkit

import (
	"context"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v9"
)

// configures the Trusted proxies list
func ConfigRouter(router *gin.Engine, trustedProxy []string) {

	if os.Getenv("GIN_MODE") == "release" {
		// turn on proxy support
		ErrorFatal("Router", router.SetTrustedProxies(trustedProxy))
	} else {
		// turn off proxy support for debugging
		ErrorFatal("Router", router.SetTrustedProxies(nil))
	}
}

// returns a static health message
func GetHealth(c *gin.Context) {
	c.Data(http.StatusOK, "application/json", []byte(`{"status":"ok"}`))
}

// redirects to the given url
func Redirect(url string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Redirect(http.StatusPermanentRedirect, url)
	}
}

// starts the router on the given IP and port
func StartRouter(router *gin.Engine, bind string) {
	if err := router.Run(bind); err != nil {
		ErrorFatal("FoxKit", err)
	}
}

// returns true, if the client requested json format, also sets the response to 406, if not
func JsonRequested(c *gin.Context, appName string) bool {
	if c.GetHeader("Content-Type") != "application/json" {
		c.AbortWithStatus(http.StatusNotAcceptable)
		LogEvent(appName, "Received request with wrong Content-Type header")
		return false
	}
	return true
}

// bind the received json to the given struct, sets the status to 400 if false
func BindJson(c *gin.Context, obj any, appName string) bool {
	if err := c.BindJSON(obj); err != nil {
		c.AbortWithStatus(http.StatusBadRequest)
		LogError(appName, err)
		return false
	}
	return true
}

// returns true if an error happened, sets the status to 500 if true
func CheckError(c *gin.Context, err *error, appName string) bool {
	if *err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		LogError(appName, *err)
		return true
	}
	return false
}

// checks if the password matches the hash, sets the HTTP code to 500 on DB failure / 401 if the password is wrong
func CheckPassword(c *gin.Context, hash, password *string) bool {
	match, err := ComparePasswordAndHash(password, hash)
	if CheckError(c, &err, "FoxKit") {
		return false
	} else if !match {
		c.AbortWithStatus(http.StatusUnauthorized)
		LogEvent("FoxKit", "loginUser(): Invalid password received")
		return false
	}
	return true
}

// checks if the session is valid, sets the HTTP code to 500 on DB error / 401 on non-valid if false
func CheckSession(ctx *context.Context, c *gin.Context, userID, token *string, redisClient *redis.Client, sessionTime time.Duration) bool {
	valid, err := ValidateSession(ctx, userID, token, redisClient, sessionTime)
	if CheckError(c, &err, "FoxKit") {
		c.AbortWithStatus(http.StatusInternalServerError)
		LogError("FoxKit", err)
		return false
	} else if !valid {
		c.AbortWithStatus(http.StatusUnauthorized)
		LogEvent("FoxKit", "Received invalid session")
		return false
	}
	return true
}

// checks if both token match, sets the HTTP code to 400 on encoding failure / 401 on token miss match
func CheckToken(c *gin.Context, stringOne, stringTwo *string) bool {
	match, err := RandomStringCompare(stringOne, stringTwo)
	if err != nil {
		c.AbortWithStatus(http.StatusBadRequest)
		LogError("FoxKit", err)
		return false
	} else if !match {
		c.AbortWithStatus(http.StatusUnauthorized)
		LogEvent("FoxKit", "Received invalid Token")
		return false
	}
	return true
}
