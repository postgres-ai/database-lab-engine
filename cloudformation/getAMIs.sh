#!/bin/bash
# This script takes a a parameter which needs to be a name of an AWS AMI
# The string will have to identify the AMI uniquely in all regions.
# The script will generate an output which can be copied into json files of AWS CloudFormation
#
# The script uses the AWS command line tools.
# The AWS command line tools have to have a default profile with the permission to
# describe a region and to describe an image

# The script can be run with normal OS user privileges.
# The script is not supposed to modify anything.
# There is no warranty. Please check the script upfront. You will use it on your own risk

# String to be used when no AMI is available in region
NOAMI="NOT_SUPPORTED"

# Change your aws profile if needed here (example, " --profile default"):
PROFILE=${DLE_AWS_PROFILE:-""}

DLE_CF_TEMPLATE_FILE="${DLE_CF_TEMPLATE_FILE:-dle_cf.yaml}"

# Check whether AWS CLI is installed and in search path
if ! aws_loc="$(type -p "aws")" || [ -z "$aws_loc" ]; then
echo "Error: Script requires AWS CLI . Install it and retry"
exit 1
fi

# Check whether parameter has been provided
if [ -z "$1" ]
then
NAME=DBLABserver*
echo "No parameter provided."
else
NAME=$1
fi
echo "Will search for AMIs with name: ${NAME}"
echo "---------------------------------------"

# Clean AWSRegionArch2AMI list
yq e -i 'del( .Mappings.AWSRegionArch2AMI )' ${DLE_CF_TEMPLATE_FILE}

##NAME=DBLABserver*
Regions=$(aws ec2 describe-regions --query "Regions[].{Name:RegionName}" --output text ${PROFILE})
for i in $Regions; do
  AMI=`aws ec2 describe-images --owners 005923036815 --region $i --filters "Name=name,Values=${NAME}" --output json | jq -r '.Images | sort_by(.CreationDate) | last(.[]).ImageId'`
  if [ -z "$AMI" ]; then
    AMI=$NOAMI
  fi

  echo  "    "${i}: $'\n' "     "HVM64: ${AMI}

  if [ "$AMI" != "$NOAMI" ]; then
    yq e -i ".Mappings.AWSRegionArch2AMI.\"${i}\" = {\"HVM64\": \"${AMI}\"}" ${DLE_CF_TEMPLATE_FILE}
  fi
done

