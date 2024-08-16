# Clickable AI

This is an experiment in making LLMs browsable and clickable instead of promptable and searchable.

* Starts a web server where keywords and generated technical documentation is presented.
* Click a keyword to delve deeper into that topic.
* Select text to make it into a button that can be clicked to delve deeper into that topic.

Uses Gemini 1.5 Flash.

Set the `PROJECT_ID` environment variable to your Google Cloud Project and also remember to log in with `gcloud auth application-default login` if you want to test this locally.

### General info

* License: Apache 2
* Version: 0.2.2
