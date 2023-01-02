package main

import (
    "bytes"
    "encoding/json"
    "fmt"
    "os"
    "text/template"
    "time"
)

type Credentials struct {
    ClientID     string   `json:"client_id"`
    ClientSecret string   `json:"client_secret"`
    Delay        Duration `json:"delay"`
}

type Command struct {
    args []*template.Template
}

type Path struct {
    path *template.Template
}

type Channel struct {
    Name            string
    DownloadCmd     *Command
    DownloadDir     *Path
    PostDownloadCmd *Command //nil = use default
    LogFile         *Path
}

type Duration struct {
    time.Duration
}

type Config struct {
    Apps            []Credentials `json:"apps"`
    Channels        []Channel     `json:"channels"`
    Delay           Duration      `json:"delay"`
    DownloadCmd     *Command      `json:"download_command"`
    DownloadDir     *Path         `json:"download_directory"`
    PostDownloadCmd *Command      `json:"post_download_command"`
    LogFile         *Path         `json:"log_file"`
}

func parseTemplate(name, tmpl string) (*template.Template, error) {
    return template.New(name).Option("missingkey=error").Parse(tmpl)
}

func LoadConfig(path string) (*Config, error) {
    data, err := os.ReadFile(path)
    if err != nil {
        return nil, err
    }
    var cfg Config
    if err := json.Unmarshal(data, &cfg); err != nil {
        return nil, err
    }

    if cfg.DownloadCmd == nil {
        cfg.DownloadCmd = &Command {
            args: []*template.Template {
                template.Must(parseTemplate("argument", "yt-dlp")),
                template.Must(parseTemplate("argument", "--parse-metadata")),
                template.Must(parseTemplate("argument", "%(release_date,upload_date,epoch>%Y%m%d)s:%(date)s")),
                template.Must(parseTemplate("argument", "--parse-metadata")),
                template.Must(parseTemplate("argument", "%(release_date,upload_date,epoch>%Y%m%d)s:%(meta_date)s")),
                template.Must(parseTemplate("argument", "--parse-metadata")),
                template.Must(parseTemplate("argument", "%(fulltitle,title)s:%(meta_title)s")),
                template.Must(parseTemplate("argument", "-o")),
                template.Must(parseTemplate("argument", "%(uploader_id)s-%(date)s-%(id)s.%(ext)s")),
                template.Must(parseTemplate("argument", "--write-info-json")),
                template.Must(parseTemplate("argument", "--write-thumbnail")),
                template.Must(parseTemplate("argument", "--embed-metadata")),
                template.Must(parseTemplate("argument", "--embed-thumbnail")),
                template.Must(parseTemplate("argument", "https://twitcasting.tv/{{.Broadcaster.ScreenID}}/movie/{{.Movie.ID}}")),
            },
        }
    }
    if cfg.DownloadDir == nil {
        cfg.DownloadDir = &Path {
            path: template.Must(parseTemplate("path", "recordings")),
        }
    }
    if cfg.LogFile == nil {
        cfg.LogFile = &Path {
            path: template.Must(parseTemplate("path", "{{.Broadcaster.ScreenID}}-{{.Movie.ID}}.log")),
        }
    }

    if cfg.Delay.Duration == 0 {
        cfg.Delay.Duration = time.Second
    }

    for i, _ := range cfg.Apps {
        a := &cfg.Apps[i]
        if a.Delay.Duration == 0 {
            a.Delay = cfg.Delay
        }
    }

    for i, _ := range cfg.Channels {
        c := &cfg.Channels[i]
        if c.DownloadCmd == nil {
            c.DownloadCmd = cfg.DownloadCmd
        }
        if c.DownloadDir == nil {
            c.DownloadDir = cfg.DownloadDir
        }
        if c.PostDownloadCmd == nil {
            c.PostDownloadCmd = cfg.PostDownloadCmd
        }
        if c.LogFile == nil {
            c.LogFile = cfg.LogFile
        }
    }
    return &cfg, nil
}

func (c *Command) Format(resp *CurrentLiveResponse) ([]string, error) {
    var args []string
    for _, t := range c.args {
        var out bytes.Buffer
        if err := t.Execute(&out, resp); err != nil {
            return nil, err
        }
        args = append(args, out.String())
    }
    return args, nil
}

func (c *Command) UnmarshalJSON(data []byte) error {
    var parts []string
    if err := json.Unmarshal(data, &parts); err != nil {
        return err
    }
    var args []*template.Template
    for _, v := range parts {
        t, err := parseTemplate("argument", v)
        if err != nil {
            return fmt.Errorf("Unable to parse template '%s': %w", v, err)
        }
        args = append(args, t)
    }
    *c = Command {
        args: args,
    }
    return nil
}

func (d *Path) Format(resp *CurrentLiveResponse) (string, error) {
    var out bytes.Buffer
    if err := d.path.Execute(&out, resp); err != nil {
        return "", err
    }
    return out.String(), nil
}

func (d *Path) UnmarshalJSON(data []byte) error {
    var s string
    if err := json.Unmarshal(data, &s); err != nil {
        return err
    }

    t, err := parseTemplate("path", s)
    if err != nil {
        return fmt.Errorf("Unable to parse template '%s': %w", err)
    }
    *d = Path {
        path: t,
    }
    return nil
}

func (c *Channel) UnmarshalJSON(data []byte) error {
    var name string
    if err := json.Unmarshal(data, &name); err == nil {
        *c = Channel {
            Name: name,
        }
        return nil
    }
    d := struct {
        Name            string   `json:"name"`
        DownloadCmd     *Command `json:"download_command"`
        DownloadDir     *Path    `json:"download_directory"`
        PostDownloadCmd *Command `json:"post_download_command"`
        LogFile         *Path    `json:"log_file"`
    } {}
    if err := json.Unmarshal(data, &d); err != nil {
        return err
    }
    *c = Channel {
        Name:            d.Name,
        DownloadCmd:     d.DownloadCmd,
        DownloadDir:     d.DownloadDir,
        PostDownloadCmd: d.PostDownloadCmd,
        LogFile:         d.LogFile,
    }
    return nil
}

func (d *Duration) UnmarshalJSON(data []byte) error {
    var v interface{}
    if err := json.Unmarshal(data, &v); err != nil {
        return err
    }
    switch value := v.(type) {
    case float64:
        d.Duration = time.Duration(value) * time.Second
        return nil
    case string:
        var err error
        d.Duration, err = time.ParseDuration(value)
        if err != nil {
            return err
        }
        return nil
    default:
        return fmt.Errorf("invalid duration '%v'", v)
    }
}

