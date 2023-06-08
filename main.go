package main

import (
	"context"
	"fmt"
	"net/url"
	"os"

	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/vartanbeno/go-reddit/v2/reddit"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type NewsItem struct {
	Title    string
	URL      *url.URL
	Score    int
	Comments int
}

func main() {
	bootstrap()

	mainApp := app.New()
	mainWindow := mainApp.NewWindow("Tinder for Go News")
	mainCanvas := mainWindow.Canvas()

	leftArrowIcon := widget.NewIcon(theme.NavigateBackIcon())
	rightArrowIcon := widget.NewIcon(theme.NavigateNextIcon())
	leftLabel := widget.NewLabel("Swipe left for boring news")
	rightLabel := widget.NewLabel("Swipe right for interesting news")

	instructions := container.NewHBox(leftArrowIcon, leftLabel, rightLabel, rightArrowIcon)

	infinite := widget.NewProgressBarInfinite()
	infinite.Start()
	firstLoad := true

	newsContainer := container.NewVBox()
	newsContainer.Add(infinite)

	mainContainer := container.NewVBox(instructions, newsContainer)
	mainCanvas.SetContent(mainContainer)

	newsItems := make(chan NewsItem)
	go fetchNewsItems(newsItems)

	// Create a goroutine to handle the news items
	go func() {
		for {
			newsItem := <-newsItems

			hyperlinkToNewsItem := widget.NewHyperlink("Go to item", newsItem.URL)
			newsCard := widget.NewCard(
				newsItem.Title,
				fmt.Sprintf("Score: %d | Comments: %d", newsItem.Score, newsItem.Comments),
				hyperlinkToNewsItem,
			)
			newsContainer.Add(newsCard)
			if firstLoad {
				firstLoad = false
				newsContainer.Remove(infinite)
			}
			mainCanvas.Refresh(mainContainer)
		}
	}()

	mainWindow.ShowAndRun()
}

func bootstrap() {
	zerolog.SetGlobalLevel(zerolog.InfoLevel)

	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
}

const defaultRedditNewsItemsWeeklyCount = 7

func fetchNewsItems(newsItems chan NewsItem) {
	client, err := reddit.NewReadonlyClient()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create Reddit client")
	}
	posts, _, err := client.Subreddit.TopPosts(context.Background(), "golang", &reddit.ListPostOptions{
		ListOptions: reddit.ListOptions{
			Limit: defaultRedditNewsItemsWeeklyCount,
		},
		Time: "week",
	})
	if err != nil {
		panic(err)
	}
	log.Debug().Int("count", len(posts)).Msg("Received posts")
	for _, post := range posts {
		url, urlParseErr := url.Parse("https://www.reddit.com" + post.Permalink)
		if urlParseErr == nil {
			log.Debug().
				Str("title", post.Title).
				Str("url", post.URL).
				Int("score", post.Score).
				Int("comments", post.NumberOfComments).
				Msg("Sending news item")
			newsItems <- NewsItem{
				Title:    post.Title,
				URL:      url,
				Score:    post.Score,
				Comments: post.NumberOfComments,
			}
		} else {
			log.Error().
				Err(urlParseErr).
				Str("title", post.Title).
				Str("url", post.URL).
				Int("score", post.Score).
				Int("comments", post.NumberOfComments).
				Msg("Failed to parse URL")
		}
	}
}
