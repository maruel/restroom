# restroom: query when people are looking at twitter

This stupid app was brought to you by this tweet:

![](https://raw.githubusercontent.com/wiki/maruel/restroom/toilet_only.png)

## Usage

restroom keeps a local cache in `restroom.json`. First generate it with:

    restroom -k <consumerkey> -c <consumersecret> -t <token> -s <tokensecret> -u <user> -v

then you can rerun it without fetching with:

    restroom -u <user>
