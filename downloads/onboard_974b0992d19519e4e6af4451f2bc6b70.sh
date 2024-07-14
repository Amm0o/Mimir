#!/bin/bash
	# Define variables
	tenantID=6a63b790-eead-4e12-869c-2ca3a9da650d
	deviceID=974b0992d19519e4e6af4451f2bc6b70

	# Ensure the target directory exists
	mkdir -p /opt/cloud-vigilante

	# Generate the JSON object and store it
	echo "{\"tenantID\": \"$tenantID\", \"deviceID\": \"$deviceID\"}" > /opt/cloud-vigilante/cloudVigilanteOnboarding.json

	echo "JSON object stored successfully."