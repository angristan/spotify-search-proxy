package main

// import (
// 	"context"
// 	"time"

// 	"github.com/sirupsen/logrus"
// 	"github.com/zmb3/spotify/v2"
// 	spotifyauth "github.com/zmb3/spotify/v2/auth"
// 	"golang.org/x/oauth2/clientcredentials"
// )

// // Check if the token expires soon, and if so recreates an API client with a new token
// func renewToken(ctx context.Context) {
// 	ticker := time.NewTicker(time.Second * 10)
// 	defer ticker.Stop()

// 	for range ticker.C {
// 		logrus.Info("Checking if Spotify token needs to be renewed")

// 		spotifyToken, err := APIClient.Token()
// 		if err != nil {
// 			logrus.WithError(err).Error("Failed to get token")
// 		}
// 		if time.Until(spotifyToken.Expiry) > time.Minute*5 {
// 			logrus.Info("Token is still valid, no need to refresh")
// 			return
// 		}

// 		spotifyConfig := &clientcredentials.Config{
// 			ClientID:     GetEnv().SpotifyClientID,
// 			ClientSecret: GetEnv().SpotifyClientSecret,
// 			TokenURL:     spotifyauth.TokenURL,
// 		}
// 		token, err := spotifyConfig.Token(ctx)
// 		if err != nil {
// 			logrus.WithError(err).Error("Failed to refresh token")
// 			return
// 		}

// 		httpClient := spotifyauth.New().Client(ctx, token)
// 		APIClient = spotify.New(httpClient)

// 		logrus.Info("Token refreshed")
// 	}
// }
