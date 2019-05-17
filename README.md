# Application Setup

Install Go on your user profile:
```
$ sudo dnf install golang
```

Then, assuming that Go is installed in the `~/go` directory, clone `https://github.com/RedHatInsights/platform-go-middlewares` into this folder:
```
~/go/src/github.com/RedHatInsights/platform-go-middlewares
```

Once that completes, clone this repo into the folder:
```
~/go/src/github.com/RedHatInsights/entitlements-api-go
```

Then, install the project's Go dependencies by running:
```
sudo sh /scripts/dev_deps.sh
```

# Certificates and Configuration

## Getting an Enterprise Cert

To run the Entitlements API locally, you will need an Enterprise Services cert with access to the dev subscription endpoint /search/criteria.

* You can request a personal cert by following ALL steps in this [mojo doc](https://mojo.redhat.com/docs/DOC-1144091).
* You should be emailed a link that will allow you to import your pk12 cert into Firefox.

After importing the pk12 cert into Firefox, you can export it into a separate .crt and .key file that can be used to
query subscription services. To export your .crt and .key file:

* Export your pk12 cert to your local box:
  * Go to Firefox preferences
  * Select Privacy & Security
  * Select View Certificates
  * Select your pk12 cert
  * Select Backup...
  * Save as a pk12 file  
* From here you can export your crt and key like so:
    `openssl pkcs12 -in your-p12-cert.p12 -out your-key.key -nocerts -nodes`
    `openssl pkcs12 -in your-p12-cert.p12 -out your-cert-sans-key.crt -clcerts -nokeys`

## Create your config file

You'll need to make a config file specific to your machine. Create a file `/config/development.{your_user_name}.sh` and enter the following:

```
export ENT_KEY=/{path_to_key}.key
export ENT_CERT=/{path_to_cert}.crt
export ENT_PORT=3000
export ENT_SUBSHOST=https://subscription.api.redhat.com
```

Replace {path_to_key} and {path_to_cert} with the locations of the .key and .crt files from the previous section.

# Running the Application

Now that everything is set up, you can run the application using:
```
sh scripts/watch.sh ./config/development.{your_user_name}.sh
```
