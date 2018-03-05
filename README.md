# Archivr
an overly simplistic tumblr archiver. I made this so I could download my blog as a set of html files and host them on a "free" static hosting provider.

## Usage
archivr is a command line app and it uses the following command line args:

| flag | desc | required? |
| ---- |----| ----|
| -key | your tumblr oauth consumer key from [here](https://www.tumblr.com/oauth/apps) | yes |
| -blog | the name (subdomain or custom domain name) of the blog you want to archive | yes |
| -o | the output directory, defaults to [blogname]-archive | no |

it will save each post as an html file named like the slug provided by the tumblr api. I don't care about file extensions, but your hoster may: github pages does, for example.

## TODO
 * download images
 * handle post types other than text