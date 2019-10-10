# Entitlements Service

## Application Setup

Install Golang:

```sh
sudo dnf install golang # or brew install go on OSX
```

Clone this repo:

```sh
git clone git@github.com:RedHatInsights/entitlements-api-go.git
```

Then, install the project's Go dependencies by running:

```sh
bash ./scripts/dev_deps.sh
```

## Certificates and Configuration

### Getting an Enterprise Cert

To run the Entitlements API locally, you will need an Enterprise Services cert with access to the dev subscription endpoint /search/criteria.

* You can request a personal cert by following ALL steps in this [mojo doc](https://mojo.redhat.com/docs/DOC-1144091).
* You can export your crt and key like so:  
    `openssl pkcs12 -in your-p12-cert.p12 -out your-key.key -nocerts -nodes`  
    `openssl pkcs12 -in your-p12-cert.p12 -out your-cert-sans-key.crt -clcerts -nokeys`

### Create your config file

You'll need to make a config file specific to your machine.
Create a local config directory: `mkdir -p ./local`
Add a file that contains your local configuration options: `$EDITOR ./local/qa.conf.sh`
The contents should look like this:

```sh
export ENT_KEY=/{path_to_key}.key
export ENT_CERT=/{path_to_cert}.crt
export ENT_SUBS_HOST=https://subscription.qa.api.redhat.com
```

Replace {path_to_key} and {path_to_cert} with the locations of the .key and .crt files from the previous section.

## Running the Application

Now that everything is set up, you can run the application using:

```bash
bash ./scripts/watch.sh ./local/development.env.sh
```

## Testing Entitlements API with curl

The Entitlements API requires that you pass in a valid x-redhat-identity header or it rejects requests.
For an example see `cat ./scripts/xrhid_helper.sh`

## Running the Unit Tests

* To run the unit tests, execute the following commands from the terminal:
    `cd controllers`
    `go test`
