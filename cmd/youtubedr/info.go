package main

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
)

// Define two new struct in local scope
type VideoFormat struct {
	Itag         int
	VideoQuality string
	AudioQuality string
	Size         int64
	Bitrate      int
	MimeType     string
}

type VideoInfo struct {
	Title       string
	Author      string
	Duration    string
	Description string
	Formats     []VideoFormat
}

type outputFunction func(VideoInfo, io.Writer) error

var outputWriters = map[string]outputFunction{
	"json": func(info VideoInfo, w io.Writer) error {
		encoder := json.NewEncoder(w)
		encoder.SetIndent("", "  ")
		return encoder.Encode(info)
	},
	"xml": func(info VideoInfo, w io.Writer) error {
		return xml.NewEncoder(w).Encode(info)
	},
	"plain": func(info VideoInfo, w io.Writer) error {
		fmt.Println("Title:      ", info.Title)
		fmt.Println("Author:     ", info.Author)
		fmt.Println("Duration:   ", info.Duration)
		fmt.Println("Description:", info.Description)
		fmt.Println()

		table := tablewriter.NewWriter(os.Stdout)
		table.SetAutoWrapText(false)
		table.SetHeader([]string{"itag", "video quality", "audio quality", "size [MB]", "bitrate", "MimeType"})

		for _, format := range info.Formats {
			table.Append([]string{
				strconv.Itoa(format.Itag),
				format.VideoQuality,
				format.AudioQuality,
				fmt.Sprintf("%0.1f", float64(format.Size)/1024/1024),
				strconv.Itoa(format.Bitrate),
				format.MimeType,
			})
		}

		table.Render()
		return nil
	},
}

// infoCmd represents the info command
var infoCmd = &cobra.Command{
	Use:   "info",
	Short: "Print metadata of the desired video",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		outputWriter := outputWriters[outputFormat]
		if outputWriter == nil {
			fmt.Fprintln(os.Stderr, "invalid format:", outputFormat)
			os.Exit(1)
		}

		video, err := getDownloader().GetVideo(args[0])
		exitOnError(err)

		videoInfo := VideoInfo{
			Title:       video.Title,
			Author:      video.Author,
			Duration:    video.Duration.String(),
			Description: video.Description,
		}

		for _, format := range video.Formats {
			bitrate := format.AverageBitrate
			if bitrate == 0 {
				// Some formats don't have the average bitrate
				bitrate = format.Bitrate
			}

			size, _ := strconv.ParseInt(format.ContentLength, 10, 64)
			if size == 0 {
				// Some formats don't have this information
				size = int64(float64(bitrate) * video.Duration.Seconds() / 8)
			}

			videoInfo.Formats = append(videoInfo.Formats, VideoFormat{
				Itag:         format.ItagNo,
				VideoQuality: format.QualityLabel,
				AudioQuality: strings.ToLower(strings.TrimPrefix(format.AudioQuality, "AUDIO_QUALITY_")),
				Size:         size,
				Bitrate:      bitrate,
				MimeType:     format.MimeType,
			})
		}

		exitOnError(outputWriter(videoInfo, os.Stdout))
	},
}

var outputFormat string

func init() {
	rootCmd.AddCommand(infoCmd)
	infoCmd.Flags().StringVarP(&outputFormat, "format", "f", "plain", "The output format (plain/json/xml).")
}