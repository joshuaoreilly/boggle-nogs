# Boggle Nogs

Boggle Nogs is an alternative front-end to Hacker News ([news.ycombinator.com](news.ycombinator.com)) which allows for ignoring submissions from certain websites or with titles containing specified strings.
You can find a live instance [here](https://bn.joshuaoreilly.com/).

## Motivation

I really like Hacker News.
It's a great resource for wacky and insightful engineering and tomfoolery.
I find myself getting sidetracked perhaps too often with the news articles that are submitted.
They tend to garner a lot of attention and make it to the front page, and I don't leave the comment section feeling any better off (it's also telling that I don't read the articles myself).

So, Boggle Nogs, a front-end that can filter out content that I would otherwise find myself reading but not really getting anything out of.

And for you, it's whatever you'd like it to be.
Ignore submissions from your arch-nemesis, shun the latest developments in the impending world takeover by frogs, or add `.*` to your `ignore-titles.txt` file for the most sophisticated experience.

## Requirements

[Go](https://go.dev/); I'm currently using version 1.19, but I suspect it'll work with older or newer versions.

## Building

Clone the repo, open the main folder, and run `go build`.

## Usage

### Running Boggle Nogs

Run:

```
./boggle-nogs
```

### Command line arguments

`--port`: specify the port to run on

`--ignore`: specify folder which contains `ignore-sites.txt` and `ignore-titles.txt`. If not given, will assume the current directory is where they are found.

There are technically other arguments, but considering I don't think they do anything or work, I'd just ignore them.

### Updating ignore/block lists

To ignore submissions from a given website, add the website's base url on a new line in `ignore-sites.txt`.
For example, to ignore all submissions from my website:

```
... (other site domains)
joshuaoreilly.com
```

To ignore submissions containing a specific string in their title, add the string on a new line in `ignore-titles.txt`.
For example, to contain all submission titles which contain the word "Frog":

```
... (other title strings)
Frog
```

Boggle Nogs uses RegEx to match sites and titles; if you'd like to ignore submissions from "oreilly.com" but not from "joshuaoreilly.com", put the following in `ignore-sites.txt`:

```
^oreilly.com$
```

Likewise, for ignoring all submission titles that contain the word "Frog", regardless of their capitalization, put the following in `ignore-titles.txt`:

```
(?i)frog
```

#### Docker

Run:

```
docker run -p 8080:8080 -v /path/to/ignore/files:/ignore boggle-nogs:latest
```

Or with docker-compose:

```
version: "3.0"

services:
  boggle-nogs:
    image: oreillyjoshua/boggle-nogs:1.0.0
    container_name: boggle-nogs
    restart: unless-stopped
    ports:
      - "8080:8080"
    volumes:
      - /path/to/ignore/folder:/ignore
```

## TODO

- [ ] Update docker hub README
- [ ] Allow for head.html, etc. to be stored somewhere else
- [ ] Create function to handle creating regex to avoid repetitive code
- [x] Build docker image
- [x] Decide on and standardize ignore-naming (is it ignore? ignored? blocked? etc.)
- [x] Grey text alternative with better background contrast
