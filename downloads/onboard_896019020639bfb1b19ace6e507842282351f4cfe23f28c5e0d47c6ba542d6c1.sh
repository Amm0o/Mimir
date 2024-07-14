#!/bin/bash
	# Define variables
	tenantID=6a63b790-eead-4e12-869c-2ca3a9da650d
	deviceID=896019020639bfb1b19ace6e507842282351f4cfe23f28c5e0d47c6ba542d6c1

	# Ensure the target directory exists
	mkdir -p /opt/cloud-vigilante

	# Generate the JSON object and store it
	echo "{\"tenantID\": \"$tenantID\", \"deviceID\": \"$deviceID\"}" > /opt/cloud-vigilante/cloudVigilanteOnboarding.json

	echo "JSON object stored successfully."