// Package steamsearch gets information about steam games from steam API
package steamsearchgo

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

const appSearchUriBase = "https://steamcommunity.com/actions/SearchApps/"
const appReviewsUri = "https://store.steampowered.com/appreviews/"
const appDetailsUri = "https://store.steampowered.com/api/appdetails?appids="

type searchResult struct {
	Appid string `json:"appid"`
	Name  string `json:"name"`
	Icon  string `json:"icon"`
	Logo  string `json:"logo"`
}

// Steam app details JSON schema
type appDetails struct {
	Success bool `json:"success"`
	Data    struct {
		Type                string   `json:"type,omitempty"`
		Name                string   `json:"name,omitempty"`
		SteamAppid          int      `json:"steam_appid,omitempty"`
		RequiredAge         int      `json:"required_age,omitempty"`
		IsFree              bool     `json:"is_free,omitempty"`
		Dlc                 []int    `json:"dlc,omitempty"`
		DetailedDescription string   `json:"detailed_description,omitempty"`
		AboutTheGame        string   `json:"about_the_game,omitempty"`
		ShortDescription    string   `json:"short_description,omitempty"`
		SupportedLanguages  string   `json:"supported_languages,omitempty"`
		HeaderImage         string   `json:"header_image,omitempty"`
		CapsuleImage        string   `json:"capsule_image,omitempty"`
		CapsuleImagev5      string   `json:"capsule_imagev5,omitempty"`
		Website             string   `json:"website,omitempty"`
		Developers          []string `json:"developers,omitempty"`
		Publishers          []string `json:"publishers,omitempty"`
		PackageGroups       []any    `json:"package_groups,omitempty"`
		Platforms           struct {
			Windows bool `json:"windows,omitempty"`
			Mac     bool `json:"mac,omitempty"`
			Linux   bool `json:"linux,omitempty"`
		} `json:"platforms,omitempty"`
		Categories []struct {
			ID          int    `json:"id,omitempty"`
			Description string `json:"description,omitempty"`
		} `json:"categories,omitempty"`
		Genres []struct {
			ID          string `json:"id,omitempty"`
			Description string `json:"description,omitempty"`
		} `json:"genres,omitempty"`
		Achievements struct {
			Total       int `json:"total,omitempty"`
			Highlighted []struct {
				Name string `json:"name,omitempty"`
				Path string `json:"path,omitempty"`
			} `json:"highlighted,omitempty"`
		} `json:"achievements,omitempty"`
		ReleaseDate struct {
			ComingSoon bool   `json:"coming_soon,omitempty"`
			Date       string `json:"date,omitempty"`
		} `json:"release_date,omitempty"`
		SupportInfo struct {
			URL   string `json:"url,omitempty"`
			Email string `json:"email,omitempty"`
		} `json:"support_info,omitempty"`
		ContentDescriptors struct {
			Ids   []int  `json:"ids,omitempty"`
			Notes string `json:"notes,omitempty"`
		} `json:"content_descriptors,omitempty"`
		Ratings struct {
			Dejus struct {
				Rating      string `json:"rating,omitempty"`
				Descriptors string `json:"descriptors,omitempty"`
				UseAgeGate  string `json:"use_age_gate,omitempty"`
				RequiredAge string `json:"required_age,omitempty"`
			} `json:"dejus,omitempty"`
		} `json:"ratings,omitempty"`
	} `json:"data,omitempty"`
}

// Stream Review JSON schema
type reviewInfo struct {
	Success      int `json:"success"`
	QuerySummary struct {
		NumReviews      int    `json:"num_reviews"`
		ReviewScore     int    `json:"review_score"`
		ReviewScoreDesc string `json:"review_score_desc"`
		TotalPositive   int    `json:"total_positive"`
		TotalNegative   int    `json:"total_negative"`
		TotalReviews    int    `json:"total_reviews"`
	} `json:"query_summary"`
}

func GetAppReview(searchTxt string) (string, string, error) {

	// Get search results from API
	response, err := steamAppSearch(searchTxt)
	if err != nil {
		fmt.Println(err)
		return "", "", err
	}

	// Get App
	app, err := findAppId(searchTxt, response)
	if err != nil {
		fmt.Println(err)
		return "", "", err
	}

	reviewBlurb, err := getReviewInfo(app)
	if err != nil {
		fmt.Println(err)
		return "", "", err
	}

	return reviewBlurb, app.Appid, nil

}

func CheckAppIsAdult(appId string) (bool, error) {

	// Create wrapper for details
	var wrapper map[int]appDetails
	// Get App details from API
	Uri := appDetailsUri + appId

	resp, err := http.Get(Uri)
	if err != nil {
		fmt.Println("failed to get app details")
		return false, err
	}

	body, _ := io.ReadAll(resp.Body)
	defer resp.Body.Close()

	jerr := json.Unmarshal(body, &wrapper)
	if jerr != nil {
		fmt.Println(jerr)
		return false, jerr
	}

	for _, v := range wrapper {
		fmt.Println("Content Ids: " + fmt.Sprint(v.Data.ContentDescriptors.Ids))
		for _, cId := range v.Data.ContentDescriptors.Ids {
			if cId == 3 { // Checking for id 3, adults only sexual context
				return true, nil
			}
		}
	}

	// If we don't find anything, give the benefit of the doubt
	return false, nil
}

func getReviewInfo(info searchResult) (string, error) {
	var review reviewInfo
	appReviewUri := appReviewsUri + fmt.Sprint(info.Appid) + "?json=1"

	resp, err := http.Get(appReviewUri)
	if err != nil {
		fmt.Println(err)
	}
	body, _ := io.ReadAll(resp.Body)
	defer resp.Body.Close()

	jerr := json.Unmarshal(body, &review)
	if jerr != nil {
		return "", jerr
	}

	blurb := "Reception for " + info.Name + " is \"" + review.QuerySummary.ReviewScoreDesc + "\" recommended by "
	blurb = blurb + fmt.Sprint(review.QuerySummary.TotalPositive) + "/" + fmt.Sprint(review.QuerySummary.TotalReviews) + " reviewers.\n"
	blurb = blurb + "For more info, check out the store page:\nhttps://store.steampowered.com/app/" + fmt.Sprint(info.Appid)
	return blurb, nil

}

func steamAppSearch(name string) (results []searchResult, err error) {

	resp, err := http.Get(appSearchUriBase + url.QueryEscape(name))
	if err != nil {
		fmt.Println(err)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(err)
	}
	defer resp.Body.Close()

	jerr := json.Unmarshal(body, &results)
	if jerr != nil {
		return results, jerr
	}

	return results, nil

}

func findAppId(name string, searchResults []searchResult) (info searchResult, err error) {

	var foundApps []searchResult

	for _, v := range searchResults {
		if strings.EqualFold(v.Name, name) { // Return an exact match immediately
			fmt.Println("found exact match! " + v.Name)
			return v, nil
		} else if strings.Contains(strings.ToLower(v.Name), strings.ToLower(name)) { // Save partial matching for processing if not exact match is available
			foundApps = append(foundApps, v)
		}
	}

	// Handle partial matches
	switch {
	case len(foundApps) == 0: // Found nothing
		return info, errors.New("Sorry, I couldn't find any results matching \"" + name + "\"")
	case len(foundApps) == 1: // Found only one match.  Return as if exact match
		return foundApps[0], nil
	case len(foundApps) > 1: // Found many matches.
		var apps string
		maxMatches := 20
		for i, v := range foundApps {
			apps = apps + v.Name + "\n"
			if i == maxMatches {
				break
			}
		}
		return info, errors.New("Sorry, I found multiple results for \"" + name + "\", try one of these possible matches:\n" + apps)
	}

	return info, errors.New("unhandled exception")
}
