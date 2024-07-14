#!/bin/bash
	# Define variables
	tenantID=6a63b790-eead-4e12-869c-2ca3a9da650d
	deviceID=6f4c0b2a5e9f90ac328ec843084bed7d

	# Ensure the target directory exists
	mkdir -p /opt/cloud-vigilante

	# Generate the JSON object and store it
	echo "{\"tenantID\": \"$tenantID\", \"deviceID\": \"$deviceID\"}" > /opt/cloud-vigilante/cloudVigilanteOnboarding.json

	echo "JSON object stored successfully."