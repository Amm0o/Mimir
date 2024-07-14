#!/bin/bash
	# Define variables
	tenantID=6a63b790-eead-4e12-869c-2ca3a9da650d
	deviceID=de77b97db5eaab423c455103d2ca3395

	# Ensure the target directory exists
	mkdir -p /opt/cloud-vigilante

	# Generate the JSON object and store it
	echo "{\"tenantID\": \"$tenantID\", \"deviceID\": \"$deviceID\"}" > /opt/cloud-vigilante/cloudVigilanteOnboarding.json

	echo "JSON object stored successfully."