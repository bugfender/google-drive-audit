# Google Drive Audit tool

This is a utility to assist in the audit of Google Drive. 
It lists all files for all users in all the company's drives, together with who has access to them.

## Setup
This tool requires some setup to be able to work:

* Create a new project in Google Cloud Platform web console
  * Navigate to https://console.developers.google.com (Google Cloud Platform web console) while logged in as a G-Suite administrator within the domain to be crawled (if the user is not added within the correct domain then the correct data will not be identified).
  * Create a new project.
  * Select Application type

* Once a new project has been created, navigate to APIs&Services → OAuth consent screen.
  * Set User type to "Internal".
  * Provide the name for new application.
  * Click Save.

* Create a new service account
  * In Google Cloud Platform web console, navigate to Credentials and click Create Credentials.
  * Then, click Service account.
  * Create service account as described in Google official article.
  * On the Grant this service account access to project (optional) step, do not select any roles.
  * On the Grant users access to this service account (optional) step, do not grant any user access. Click Done.

* Create a service account key.
  * On the Service accounts section, click edit on the account you want to create a key for.
  * Click (menu) icon under Actions and select Create key.
  * In the Create private key for <Service account name> dialog, select JSON format, and download the file to a known location as it will be required later.

* Delegate domain-wide authority to the service account
  * On the Service accounts section, select your service account and click Edit.
  * Click the Show Domain-Wide Delegation link and tick the Enable G Suite Domain-wide Delegation checkbox.
  * Click Save.
  * Once completed, review the "Domain wide delegation" column for this account and make sure that the delegation enabled.
  * Click the View Client ID link.
  * Copy your Client ID, you will need it later.

* Enable Google Drive API
  * In Google Cloud Platform web console, navigate to the API Dashboard and select Enable APIs and Services (if APIs have not previously been enabled).
  * Search for Google Drive API and click Enable (or Manage).
  * Search for Admin SDK API and click Enable (or Manage).

* Allow the use of the application in Google Workspace Admin Console
  * Navigate to http://admin.google.com
  * Navigate to Security → API Controls → Manage Domain-wide Delegation within the Google admin portal.
  * Set the client name to the Client ID you copied on the previous step.
  * Set the API scopes `https://www.googleapis.com/auth/drive` and `https://www.googleapis.com/auth/admin.directory.user` and select Authorize.

Credit to https://helpcenter.netwrix.com/NDC/NDC/Config_Infrastructure/Configure_GDrive.html

## Building

```
go build
```

## Usage

```
google-drive-audit audit [flags]

Flags:
-a, --admin-email string   email address of a domain administrator
-c, --credentials string   service credentials file (obtained from Google Cloud Platform console) (default "credentials.json")
-d, --domain string        domain name to audit
-o, --output string        output file (default "-")
-q, --quiet                do not show progress
```

Example:

    google-drive-audit audit --domain=yourcompany.com --admin-email=you@yourcompany.com

## Limitations

This tool only works on Google Workspaces domains.

## Roadmap

This tool could do other useful things for an auditor:

* Copy files shared by external users (to keep in case they were un-shared)
* Remove all permissions for a given user (for example, a supplier who no longer works for us)

These features will be implemented when needed, if ever. Please feel free to contribute.


## Support

You can get support from Beenario GmbH for a fee. Contact us if you're interested.