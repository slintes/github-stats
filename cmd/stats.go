package cmd

import (
	"context"
	"flag"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/google/go-github/v26/github"
	"golang.org/x/oauth2"
)

type PR struct {
	repo  string
	nr    int
	title string
}

type Stats struct {
	month     time.Month
	ownMerged []*PR
	commented []*PR
	merged    []*PR
}

type AllStats map[time.Month]*Stats

func Run() {

	token := flag.String("token", "", "Your github token.")
	reposToInspect := flag.String("repositories", "", "Comma separated list of github repositories to inspect, in org/name format.")
	nrMonths := flag.Int("months", 2, "Nr of months to look back; 1 = current month, 2 = current + last, and so on. Defaults to 2.")

	flag.Parse()

	if token == nil || *token == "" {
		panic("missing token flag")
	}

	if reposToInspect == nil || *reposToInspect == "" {
		panic("missing repositories flag")
	}

	// init github client and get repositories
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: *token},
	)
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)
	repos, _, err := client.Repositories.List(ctx, "", &github.RepositoryListOptions{
		ListOptions: github.ListOptions{
			PerPage: 100,
		},
	})
	if err != nil {
		panic(err)
	}

	// start on 1st day of nrMonths-1 months ago
	now := time.Now()
	limit := now.AddDate(0, -(*nrMonths - 1), 0)
	limit = time.Date(limit.Year(), limit.Month(), 1, 0, 0, 0, 0, now.Location())

	allStats := make(AllStats, *nrMonths+1)

	for _, repo := range repos {
		if matchesRepo(repo, *reposToInspect) {
			fmt.Printf("repo: %v\n", *repo.Name)
			nextPage := 1
			for nextPage != 0 {
				prs, response, err := client.PullRequests.List(ctx, *repo.Owner.Login, *repo.Name, &github.PullRequestListOptions{
					State:     "all",
					Sort:      "updated",
					Direction: "desc",
					ListOptions: github.ListOptions{
						Page:    nextPage,
						PerPage: 20,
					},
				})
				if err != nil {
					panic(err)
				}

				nextPage = response.NextPage

				for _, pr := range prs {

					fmt.Printf("pr nr %v, state %v, updated %v\n", *pr.Number, *pr.State, pr.UpdatedAt.String())

					// ignore PRs older than nrMonth
					if (*pr.UpdatedAt).Before(limit) {
						nextPage = 0
						continue
					}

					statsPR := &PR{
						repo:  *repo.Name,
						nr:    *pr.Number,
						title: *pr.Title,
					}

					// find merged PRs ownMerged or merged by me
					if pr.MergedAt != nil {
						if (*pr.MergedAt).Before(limit) {
							continue
						}

						fmt.Println("  is merged")

						month := pr.MergedAt.Month()
						stats := getStatsForMonth(allStats, month)

						if *pr.User.Login == "slintes" {
							stats.ownMerged = append(stats.ownMerged, statsPR)
							fmt.Printf("    ownMerged by me: %v\n", *pr.Title)
							continue
						}

						// get PR in order to get mergedBy
						pr, _, err := client.PullRequests.Get(ctx, *repo.Owner.Login, *repo.Name, *pr.Number)
						if err != nil {
							panic(err)
						}
						if *pr.MergedBy.Login == "slintes" {
							stats.merged = append(stats.merged, statsPR)
							fmt.Printf("    merged by me: %v\n", *pr.Title)
							continue
						}
					}

					// ignore my open PRs
					if *pr.User.Login == "slintes" {
						fmt.Printf("  skipping my own open PR: %v\n", *pr.Title)
						continue
					}

					// find PRs reviewed by me
					reviewsNextPage := 1
					for reviewsNextPage != 0 {
						reviews, rResponse, err := client.PullRequests.ListReviews(ctx, *repo.Owner.Login, *repo.Name, *pr.Number, &github.ListOptions{
							Page:    reviewsNextPage,
							PerPage: 50,
						})
						if err != nil {
							panic(err)
						}
						reviewsNextPage = rResponse.NextPage
						for _, review := range reviews {
							if *review.User.Login == "slintes" {

								// nil for pending reviews
								if review.SubmittedAt == nil || (*review.SubmittedAt).Before(limit) {
									continue
								}

								month := review.SubmittedAt.Month()
								stats := getStatsForMonth(allStats, month)
								if !containsPR(stats.commented, statsPR) {
									stats.commented = append(stats.commented, statsPR)
									fmt.Printf("    reviewed by me: %v\n", *pr.Title)
								}
							}
						}
					}

					// also check normal comments
					commentsNextPage := 1
					for commentsNextPage != 0 {
						comments, cResponse, err := client.PullRequests.ListComments(ctx, *repo.Owner.Login, *repo.Name, *pr.Number, &github.PullRequestListCommentsOptions{
							Sort:      "ownMerged",
							Direction: "decs",
							ListOptions: github.ListOptions{
								Page:    commentsNextPage,
								PerPage: 50,
							},
						})
						if err != nil {
							panic(err)
						}
						commentsNextPage = cResponse.NextPage
						for _, comment := range comments {
							if *comment.User.Login == "slintes" {

								if (*comment.CreatedAt).Before(limit) {
									continue
								}

								month := comment.CreatedAt.Month()
								stats := getStatsForMonth(allStats, month)
								if !containsPR(stats.commented, statsPR) {
									stats.commented = append(stats.commented, statsPR)
									fmt.Printf("    commented by me: %v\n", *pr.Title)
								}
							}
						}
					}
				}
			}
		}
	}

	// sort and print result
	var months []int
	for m := range allStats {
		months = append(months, int(m))
	}
	sort.Ints(months)
	for _, m := range months {
		month := time.Month(m)
		fmt.Printf("month: %v\n", month.String())

		printPrs := func(prs []*PR) {
			for _, pr := range prs {
				fmt.Printf("    %+v\n", *pr)
			}
		}

		stats := allStats[month]
		fmt.Printf("  own merged: %v\n", len(stats.ownMerged))
		printPrs(stats.ownMerged)

		fmt.Printf("  merged: %v\n", len(stats.merged))
		printPrs(stats.merged)

		fmt.Printf("  reviewed: %v\n", len(stats.commented))
		printPrs(stats.commented)
	}
}

func matchesRepo(repo *github.Repository, repos string) bool {
	rs := strings.Split(repos, ",")
	for _, r := range rs {
		parts := strings.Split(r, "/")
		if len(parts) != 2 {
			panic("malformed repositories flag value")
		}
		user := parts[0]
		name := parts[1]
		if *repo.Owner.Login == user && *repo.Name == name {
			return true
		}
	}
	return false
}

func getStatsForMonth(allStats AllStats, month time.Month) *Stats {
	stats, ok := allStats[month]
	if !ok {
		stats = &Stats{
			month:     month,
			ownMerged: make([]*PR, 0),
			commented: make([]*PR, 0),
			merged:    make([]*PR, 0),
		}
		allStats[month] = stats
	}
	return stats
}

func containsPR(prs []*PR, pr *PR) bool {
	for _, thisPR := range prs {
		if reflect.DeepEqual(thisPR, pr) {
			return true
		}
	}
	return false
}
