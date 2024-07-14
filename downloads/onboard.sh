#!/bin/bash
	# Define variables
	tenantID=6a63b790-eead-4e12-869c-2ca3a9da650d
	deviceID=7ff450927d95c6db129d8f8817b874b98c59575fbcfffcafc8dbd799eb53307d

	# Ensure the target directory exists
	mkdir -p /opt/cloud-vigilante

	# Generate the JSON object and store it
	echo "{\"tenantID\": \"$tenantID\", \"deviceID\": \"$deviceID\"}" > /opt/cloud-vigilante/cloudVigilanteOnboarding.json

	echo "JSON object stored successfully."