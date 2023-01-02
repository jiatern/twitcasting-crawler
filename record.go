package main

import (
    "fmt"
    "log"
    "os"
    "os/exec"
    "path/filepath"
)

func (c *Crawler) record(channel *Channel, resp *CurrentLiveResponse) error {
    dir, err := channel.DownloadDir.Format(resp)
    if err != nil {
        return fmt.Errorf("Malformed directory template for %s: %w", channel.Name, err)
    }

    logFile, err := channel.LogFile.Format(resp)
    if err != nil {
        return fmt.Errorf("Malformed log file template for %s: %w", channel.Name, err)
    }


    downloadCmd, err := channel.DownloadCmd.Format(resp)
    if err != nil {
        return fmt.Errorf("Malformed download command template for %s: %w", channel.Name, err)
    }
    if len(downloadCmd) == 0 {
        return fmt.Errorf("Empty download command for %s", channel.Name)
    }


    var postDownloadCmd []string
    if channel.PostDownloadCmd != nil {
        var err error
        postDownloadCmd, err = channel.PostDownloadCmd.Format(resp)
        if err != nil {
            return fmt.Errorf("Malformed post download command template for %s: %w", channel.Name, err)
        }
        if len(postDownloadCmd) == 0 {
            postDownloadCmd = nil
        }
    }

    if err := os.MkdirAll(dir, 0755); err != nil {
        return fmt.Errorf("Unable to create download directory %s: %w", dir, err)
    }

    log.Printf("[%s] Running %+q in %s", channel.Name, downloadCmd, dir)

    cmd := exec.Command(downloadCmd[0], downloadCmd[1:]...)
    cmd.Dir = dir
    cmd.Stdin = nil

    var logf *os.File
    if len(logFile) > 0 {
        var err error
        logf, err = os.OpenFile(filepath.Join(dir, logFile), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
        if err != nil {
            log.Printf("[%s] Unable to open log file: %w", channel.Name, err)
        } else {
            cmd.Stdout = logf
            cmd.Stderr = logf
        }
    }

    if err := cmd.Start(); err != nil {
        return err
    }

    //wait for it to finish asynchronously
    go func() {
        err := cmd.Wait()
        if err != nil {
            log.Printf("[%s:%s] Download failed: %w", channel.Name, resp.Movie.ID, err)
            return
        }
        if postDownloadCmd == nil {
            log.Printf("[%s:%s] Download done", channel.Name, resp.Movie.ID)
            return
        }
        log.Printf("[%s:%s] Download done, running post download command", channel.Name, resp.Movie.ID)

        cmd = exec.Command(postDownloadCmd[0], postDownloadCmd[1:]...)
        cmd.Dir = dir
        cmd.Stdin = nil
        cmd.Stdout = logf
        cmd.Stderr = logf

        if err := cmd.Run(); err != nil {
            log.Printf("[%s:%s] Post download command failed: %w", channel.Name, resp.Movie.ID, err)
        } else {
            log.Printf("[%s:%s] Post download command done", channel.Name, resp.Movie.ID)
        }
    }()
    return nil
}

