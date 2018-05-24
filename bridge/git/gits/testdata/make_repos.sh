#! /bin/bash

# Makes repos for testing our git integration

set -e

base_dir=`mktemp -d`
cd $base_dir

if [[ $(git config user.email) == "" ]]; then
    git config --global user.email "nick+gittest@windmill.engineering"
    git config --global user.name "Nick Git Test"
fi

repo1_dir=$base_dir/repo1.git/
mkdir $repo1_dir
git init --bare $repo1_dir 1>/dev/null
co1_dir=$base_dir/repo1.checkout
mkdir $co1_dir
git clone $repo1_dir $co1_dir

cd $co1_dir
echo "foo in repo1" > foo.txt

git add .
git commit -am "master in repo1" 1>/dev/null
sha_master1=`git rev-parse HEAD`

git checkout -b featureA
echo "bar in repo1 featureA" > bar.txt

git add .
git commit -am "featureA in repo1" 1>/dev/null
sha_featureA1=`git rev-parse HEAD`

git push origin *:*

cd $base_dir

repo2_dir=$base_dir/repo2.git/
mkdir $repo2_dir
git init --bare $repo2_dir 1>/dev/null
co2_dir=$base_dir/repo2.checkout
mkdir $co2_dir
git clone $repo2_dir $co2_dir

cd $co2_dir
echo "foo in repo2" > foo.txt

git add .
git commit -am "master in repo2" 1>/dev/null
sha_master2=`git rev-parse HEAD`

git push origin *:*


rm -rf $co1_dir
rm -rf $co2_dir


echo "repo1=$repo1_dir"
echo "repo1_master=$sha_master1"
echo "repo1_featureA=$sha_featureA1"
echo "repo2=$repo2_dir"
echo "repo2_master=$sha_master2"
