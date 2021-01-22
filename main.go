// Copyright (c) Derek Author(s) 2017. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"github.com/google/go-github/github"

	"github.com/alexellis/derek/auth"
	"github.com/alexellis/derek/config"

	"github.com/alexellis/derek/handler"

	"github.com/alexellis/derek/types"
	"github.com/alexellis/hmac"
	"github.com/go-http-utils/logger"
)

const (
	dcoCheck              = "dco_check"
	comments              = "comments"
	deleted               = "deleted"
	prDescriptionRequired = "pr_description_required"
	hacktoberfest         = "hacktoberfest"
	noNewbies             = "no_newbies"
	releaseNotes          = "release_notes"
)

func healthEndpoint(w http.ResponseWriter, req *http.Request) {
	fmt.Fprintf(w, "ok\n")
}

func derekEndpoint(w http.ResponseWriter, req *http.Request) {
	validateHMAC := os.Getenv("VALIDATE_HMAC")
	xGitHubEvent := req.Header.Get("X-GitHub-Event")
	xHubSignature := req.Header.Get("X-Hub-Signature")
	requestRaw, err := ioutil.ReadAll(req.Body)
	if err != nil {
		log.Println("Error during body reading")
		response(w, http.StatusUnprocessableEntity, "Error during body reading")
		return
	}
	statusCode, msg := derek(requestRaw, validateHMAC, xHubSignature, xGitHubEvent)
	response(w, statusCode, msg)
}

func response(w http.ResponseWriter, statusCode int, msg string) {
	w.WriteHeader(statusCode)
	if msg != "" {
		formattedMsg := fmt.Sprintf("%v - %v", statusCode, msg)
		w.Write([]byte(formattedMsg))
	}
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/derek", derekEndpoint)
	mux.HandleFunc("/health", healthEndpoint)
	if err := http.ListenAndServe(":8080", logger.Handler(mux, os.Stdout, logger.CombineLoggerType)); err != nil {
		log.Fatal(err)
	}
}

func derek(requestRaw []byte, validateHMAC string, xHubSignature string, xGitHubEvent string) (int, string) {
	validateHmac := hmacValidation(validateHMAC)

	if validateHmac && len(xHubSignature) == 0 {
		return 422, "must provide X-Hub-Signature"
	}

	config, configErr := config.NewConfig()
	if configErr != nil {
		fmt.Println(configErr)
		return 500, "Configuration error"
	}

	if validateHmac {
		err := hmac.Validate(requestRaw, xHubSignature, config.SecretKey)
		if err != nil {
			return 500, err.Error()
		}
	}

	if err := handleEvent(xGitHubEvent, requestRaw, config); err != nil {
		return 400, err.Error()
	}
	return 200, "done!"
}

func handleEvent(eventType string, bytesIn []byte, config config.Config) error {

	switch eventType {
	case "pull_request":
		req := types.PullRequestOuter{}
		if err := json.Unmarshal(bytesIn, &req); err != nil {
			return fmt.Errorf("Cannot parse input %s", err.Error())
		}

		customer, err := auth.IsCustomer(req.Repository.Owner.Login, &http.Client{})
		if err != nil {
			return fmt.Errorf("Unable to verify customer: %s/%s", req.Repository.Owner.Login, req.Repository.Name)
		} else if customer == false {
			return fmt.Errorf("No customer found for: %s/%s", req.Repository.Owner.Login, req.Repository.Name)
		}

		log.Printf("Owner: %s, repo: %s, action: %s", req.Repository.Owner.Login, req.Repository.Name, "pull_request")

		var derekConfig *types.DerekRepoConfig
		if req.Repository.Private {
			derekConfig, err = handler.GetPrivateRepoConfig(req.Repository.Owner.Login, req.Repository.Name, req.Installation.ID, config)
		} else {
			derekConfig, err = handler.GetRepoConfig(req.Repository.Owner.Login, req.Repository.Name)
		}

		if err != nil {
			return fmt.Errorf("Unable to access maintainers file at: %s/%s\nError: %s",
				req.Repository.Owner.Login,
				req.Repository.Name,
				err.Error())
		}

		if req.Action != handler.ClosedConstant && req.PullRequest.State != handler.ClosedConstant {
			contributingURL := getContributingURL(derekConfig.ContributingURL, req.Repository.Owner.Login, req.Repository.Name)

			if handler.EnabledFeature(dcoCheck, derekConfig) {
				log.Printf("Owner: %s, repo: %s, action: %s", req.Repository.Owner.Login, req.Repository.Name, "derek:dco_check")

				handler.HandlePullRequest(req, contributingURL, config)
			}

			if handler.EnabledFeature(prDescriptionRequired, derekConfig) {
				handler.VerifyPullRequestDescription(req, contributingURL, config)
			}

			if handler.EnabledFeature(noNewbies, derekConfig) {
				isSpamPR, _ := handler.HandleFirstTimerPR(req, contributingURL, config)
				if isSpamPR {
					return nil
				}
			}

			if handler.EnabledFeature(hacktoberfest, derekConfig) {
				isSpamPR, _ := handler.HandleHacktoberfestPR(req, contributingURL, config)
				if isSpamPR {
					return nil
				}
			}
		}
		break

	case "issue_comment":
		req := types.IssueCommentOuter{}
		if err := json.Unmarshal(bytesIn, &req); err != nil {
			return fmt.Errorf("Cannot parse input %s", err.Error())
		}

		log.Printf("Owner: %s, repo: %s, action: %s", req.Repository.Owner.Login, req.Repository.Name, "issue_comment")

		customer, err := auth.IsCustomer(req.Repository.Owner.Login, &http.Client{})
		if err != nil {
			return fmt.Errorf("Unable to verify customer: %s/%s", req.Repository.Owner.Login, req.Repository.Name)
		} else if customer == false {
			return fmt.Errorf("No customer found for: %s/%s", req.Repository.Owner.Login, req.Repository.Name)
		}

		var derekConfig *types.DerekRepoConfig
		if req.Repository.Private {
			derekConfig, err = handler.GetPrivateRepoConfig(req.Repository.Owner.Login, req.Repository.Name, req.Installation.ID, config)
		} else {
			derekConfig, err = handler.GetRepoConfig(req.Repository.Owner.Login, req.Repository.Name)
		}

		if err != nil {
			return fmt.Errorf("Unable to access maintainers file at: %s/%s\nError: %s",
				req.Repository.Owner.Login,
				req.Repository.Name,
				err.Error())
		}

		if req.Action != deleted {
			if handler.PermittedUserFeature(comments, derekConfig, req.Comment.User.Login) {
				log.Printf("Owner: %s, repo: %s, action: %s", req.Repository.Owner.Login, req.Repository.Name, "derek:handle_comment")

				handler.HandleComment(req, config, derekConfig)
			}
		}
		break

	case "release":
		req := github.ReleaseEvent{}

		if err := json.Unmarshal(bytesIn, &req); err != nil {
			return fmt.Errorf("Cannot parse input %s", err.Error())
		}

		log.Printf("Owner: %s, repo: %s, action: %s", req.Repo.Owner.GetLogin(), req.Repo.GetName(), "release")

		if req.GetAction() == "created" {
			customer, err := auth.IsCustomer(req.Repo.Owner.GetLogin(), &http.Client{})
			if err != nil {
				return fmt.Errorf("unable to verify customer: %s/%s", req.Repo.Owner.GetLogin(), req.Repo.GetName())
			} else if customer == false {
				return fmt.Errorf("no customer found for: %s/%s", req.Repo.Owner.GetLogin(), req.Repo.GetName())
			}
			var derekConfig *types.DerekRepoConfig
			if req.Repo.GetPrivate() {
				derekConfig, err = handler.GetPrivateRepoConfig(req.Repo.Owner.GetLogin(), req.Repo.GetName(), int(req.Installation.GetID()), config)
				if err != nil {
					return fmt.Errorf("unable to get private repo config: %s", err)
				}
			} else {
				derekConfig, err = handler.GetRepoConfig(req.Repo.Owner.GetLogin(), req.Repo.GetName())
				if err != nil {
					return fmt.Errorf("unable to get repo config: %s", err)
				}
			}

			err = fmt.Errorf(`"release_notes" feature not enabled`)
			if handler.EnabledFeature(releaseNotes, derekConfig) {
				log.Printf("Owner: %s, repo: %s, action: %s", req.Repo.Owner.GetLogin(), req.Repo.GetName(), "derek:handle_release")

				handler := handler.NewReleaseHandler(config, int(req.Installation.GetID()))
				err = handler.Handle(req)
			}
			return err
		}
		break
	default:
		return fmt.Errorf("X-GitHub-Event want: ['pull_request', 'issue_comment'], got: " + eventType)
	}

	return nil
}

func getContributingURL(contributingURL, owner, repositoryName string) string {
	if len(contributingURL) == 0 {
		contributingURL = fmt.Sprintf("https://github.com/%s/%s/blob/master/CONTRIBUTING.md", owner, repositoryName)
	}
	return contributingURL
}

func hmacValidation(validateHMAC string) bool {
	return (validateHMAC != "false") && (validateHMAC != "0")
}
