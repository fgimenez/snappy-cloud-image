// -*- Mode: Go; indent-tabs-mode: t -*-

/*
 * Copyright (C) 2015 Canonical Ltd
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License version 3 as
 * published by the Free Software Foundation.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 *
 */

package cloud

import (
	"fmt"
	"strconv"
	"strings"
	"testing"

	"gopkg.in/check.v1"
)

const (
	testDefaultRelease   = "rolling"
	testDefaultChannel   = "edge"
	testDefaultArch      = "amd64"
	testImageVersion     = 198
	baseCompleteResponse = `| 06c12690-08ef-4a9b-aaa6-6e8249bcfef8 | ubuntu-released/ubuntu-oneiric-11.10-amd64-server-20130509-disk1.img                                 |
| 8fa0213b-e598-473f-bb33-901281063395 | smoser-cloud-images/ubuntu-hardy-8.04-amd64-server-20121003                                          |
| 56e4a037-887f-4e8c-8e9f-edad2060232b | smoser-cloud-images/ubuntu-hardy-8.04-amd64-server-20121003-ramdisk                                  |
| cc3fff76-6e86-4bab-93a4-74c45cf3d078 | smoser-cloud-images/ubuntu-hardy-8.04-amd64-server-20121003-kernel                                   |
%s
| f5eca345-3d7c-480d-a5de-3057ef1c5e82 | smoser-cloud-images/ubuntu-hardy-8.04-i386-server-20121003                                           |
| 45a240bc-f3c7-4e2f-99b5-7761dabd67c2 | smoser-cloud-images/ubuntu-hardy-8.04-i386-server-20121003-ramdisk                                   |
| 1e2111f6-7f02-4d07-bec2-229c8dd30559 | smoser-cloud-images/ubuntu-hardy-8.04-i386-server-20121003-kernel                                    |
| 47537aad-dcdb-422e-9302-2f874f88f216 | quantal-desktop-amd64                                                                                |
%s
%s
| f3618134-0151-48a2-8964-42574322fd52 | precise-desktop-amd64                                                                                |
| 762d5ce2-fbc2-4685-8d6c-71249d19df9e | ubuntu-core/devel/ubuntu-1504-snappy-core-amd64-edge-20151020-disk1.img                              |
| 08763be0-3b3d-41e3-b5b0-08b9006fc1d7 | smoser-lucid-loader/lucid-amd64-linux-image-2.6.32-34-virtual-v-2.6.32-34.77~smloader0-build0-loader |
| 842949c6-225b-4ad0-81b7-98de2b818eed | smoser-lucid-loader/lucid-amd64-linux-image-2.6.32-34-virtual-v-2.6.32-34.77~smloader0-kernel        |
| bf412075-2c8d-4753-8d19-4e502cf57d8d | None                                                                                                 |
%s
`
	baseResponse = "| 762d5ce2-fbc2-4685-8d6c-71249d19df9e | ubuntu-core/custom/ubuntu-%s-snappy-core-%s-%s-%d-disk1.img                        |"
)

type cloudSuite struct {
	subject *Client
	cli     *fakeCliCommander
}

type fakeCliCommander struct {
	execCommandCalls map[string]int
	output           string
	err              bool
}

func (f *fakeCliCommander) ExecCommand(cmds ...string) (output string, err error) {
	f.execCommandCalls[strings.Join(cmds, " ")]++
	if f.err {
		err = fmt.Errorf("exec error")
	}
	return f.output, err
}

var _ = check.Suite(&cloudSuite{})

func Test(t *testing.T) { check.TestingT(t) }

func (s *cloudSuite) SetUpSuite(c *check.C) {
	s.cli = &fakeCliCommander{}
	s.subject = NewClient(s.cli)
}

func (s *cloudSuite) SetUpTest(c *check.C) {
	s.cli.execCommandCalls = make(map[string]int)
	s.cli.output = ""
	s.cli.err = false
}

func (s *cloudSuite) TestGetLatestVersionQueriesGlance(c *check.C) {
	s.subject.GetLatestVersion(testDefaultRelease, testDefaultChannel, testDefaultArch)

	c.Assert(s.cli.execCommandCalls["openstack image list --property status=active"], check.Equals, 1)
}

func (s *cloudSuite) TestGetLatestVersionReturnsTheLatestVersion(c *check.C) {
	version := 100
	versionLine := fmt.Sprintf(baseResponse, testDefaultRelease, testDefaultArch, testDefaultChannel, version)
	versionPlusOneLine := fmt.Sprintf(baseResponse, testDefaultRelease, testDefaultArch, testDefaultChannel, version+1)
	versionPlusTwoLine := fmt.Sprintf(baseResponse, testDefaultRelease, testDefaultArch, testDefaultChannel, version+2)

	testCases := []struct {
		glanceOutput, release, channel, arch string
		expectedVersion                      int
	}{
		{fmt.Sprintf(baseCompleteResponse, "", "", "", ""), testDefaultRelease, testDefaultChannel, testDefaultArch,
			0},
		{fmt.Sprintf(baseCompleteResponse, versionLine, "", "", ""),
			testDefaultRelease, testDefaultChannel, testDefaultArch,
			version},
		{fmt.Sprintf(baseCompleteResponse, versionLine, versionPlusOneLine, "", ""),
			testDefaultRelease, testDefaultChannel, testDefaultArch,
			version + 1},
		{fmt.Sprintf(baseCompleteResponse, versionPlusOneLine, versionLine, "", ""),
			testDefaultRelease, testDefaultChannel, testDefaultArch,
			version + 1},
		{fmt.Sprintf(baseCompleteResponse, versionPlusOneLine, versionLine, "", versionPlusTwoLine),
			testDefaultRelease, testDefaultChannel, testDefaultArch,
			version + 2},
		{fmt.Sprintf(baseCompleteResponse, versionPlusOneLine, versionPlusTwoLine, versionLine, versionPlusOneLine),
			testDefaultRelease, testDefaultChannel, testDefaultArch,
			version + 2},
	}
	for _, item := range testCases {
		s.cli.output = item.glanceOutput
		ver, _ := s.subject.GetLatestVersion(item.release, item.channel, item.arch)

		c.Check(ver, check.Equals, item.expectedVersion)
	}
}

func (s *cloudSuite) TestGetLatestVersionReturnsGlanceError(c *check.C) {
	s.cli.err = true

	_, err := s.subject.GetLatestVersion(testDefaultRelease, testDefaultChannel, testDefaultArch)

	c.Assert(err, check.NotNil)
}

func (s *cloudSuite) TestGetLatestVersionReturnsVersionNumberError(c *check.C) {
	s.cli.output = "| 762d5ce2-fbc2-4685-8d6c-71249d19df9e | ubuntu-core/custom/ubuntu-rolling-snappy-core-amd64-edge-10f-disk1.img |"

	_, err := s.subject.GetLatestVersion(testDefaultRelease, testDefaultChannel, testDefaultArch)

	c.Assert(err, check.NotNil)
	c.Assert(err, check.FitsTypeOf, &strconv.NumError{})
}

func (s *cloudSuite) TestGetLatestVersionReturnsVersionNotFoundError(c *check.C) {
	s.cli.output = fmt.Sprintf(baseCompleteResponse, "", "", "", "")

	_, err := s.subject.GetLatestVersion(testDefaultRelease, testDefaultChannel, testDefaultArch)

	c.Assert(err, check.FitsTypeOf, &ErrVersionNotFound{})
	c.Assert(err.Error(), check.Equals,
		fmt.Sprintf(errVerNotFoundPattern, testDefaultRelease, testDefaultChannel, testDefaultArch))
}

func (s *cloudSuite) TestCreateCallsGlance(c *check.C) {
	path := "mypath"
	version := 100
	err := s.subject.Create(path,
		testDefaultRelease, testDefaultChannel, testDefaultArch, version)

	c.Assert(err, check.IsNil)

	imageNamePrefix := fmt.Sprintf(imageNamePrefixPattern, testDefaultRelease, testDefaultArch, testDefaultChannel)
	expectedCall := fmt.Sprintf("openstack image create --file %s %s-%d-%s", path, imageNamePrefix, version, imageNameSufix)

	c.Assert(s.cli.execCommandCalls[expectedCall], check.Equals, 1)
}

func (s *cloudSuite) TestCreateReturnsError(c *check.C) {
	s.cli.err = true

	path := "mypath"
	version := 100
	err := s.subject.Create(path,
		testDefaultRelease, testDefaultChannel, testDefaultArch, version)

	c.Assert(err, check.NotNil)
}

func (s *cloudSuite) TestGetImageID(c *check.C) {
	testCases := []struct {
		release, channel, arch string
		version                int
		expectedID             string
	}{
		{"rolling", "edge", "amd64", 100, "ubuntu-core/custom/ubuntu-rolling-snappy-core-amd64-edge-100-disk1.img"},
		{"rolling", "stable", "amd64", 10, "ubuntu-core/custom/ubuntu-rolling-snappy-core-amd64-stable-10-disk1.img"},
		{"rolling", "alpha", "amd64", 210, "ubuntu-core/custom/ubuntu-rolling-snappy-core-amd64-alpha-210-disk1.img"},
		{"1504", "edge", "amd64", 54, "ubuntu-core/custom/ubuntu-1504-snappy-core-amd64-edge-54-disk1.img"},
		{"1504", "stable", "amd64", 23, "ubuntu-core/custom/ubuntu-1504-snappy-core-amd64-stable-23-disk1.img"},
		{"1504", "alpha", "amd64", 2105, "ubuntu-core/custom/ubuntu-1504-snappy-core-amd64-alpha-2105-disk1.img"},
	}
	for _, item := range testCases {
		c.Assert(GetImageID(item.release, item.channel, item.arch, item.version), check.Equals, item.expectedID)
	}
}

func (s *cloudSuite) TestDeleteCallsCli(c *check.C) {
	testCases := []struct {
		images       []string
		expectedCall string
	}{
		{[]string{"version1", "version2"}, "openstack delete version1 version2"},
		{[]string{"version2", "version1"}, "openstack delete version2 version1"},
		{[]string{"version2", "version1", "version3", "version4"}, "openstack delete version2 version1 version3 version4"},
	}
	for _, item := range testCases {
		s.subject.Delete(item.images...)
		c.Assert(s.cli.execCommandCalls[item.expectedCall], check.Equals, 1)
	}
}

func (s *cloudSuite) TestDeleteReturnsCliError(c *check.C) {
	s.cli.err = true

	err := s.subject.Delete("image1", "image2")

	c.Assert(err, check.NotNil)
}

func (s *cloudSuite) TestGetVersionsReturnsImageNames(c *check.C) {
	version := 100
	versionLine := fmt.Sprintf(baseResponse, testDefaultRelease, testDefaultArch, testDefaultChannel, version)
	versionID := getIDFromGlanceResponse(versionLine)
	versionPlusOneLine := fmt.Sprintf(baseResponse, testDefaultRelease, testDefaultArch, testDefaultChannel, version+1)
	versionPlusOneID := getIDFromGlanceResponse(versionPlusOneLine)
	versionPlusTwoLine := fmt.Sprintf(baseResponse, testDefaultRelease, testDefaultArch, testDefaultChannel, version+2)
	versionPlusTwoID := getIDFromGlanceResponse(versionPlusTwoLine)

	testCases := []struct {
		glanceOutput, release, channel, arch string
		expectedImageNames                   []string
	}{
		{fmt.Sprintf(baseCompleteResponse, "", "", "", ""), testDefaultRelease, testDefaultChannel, testDefaultArch,
			[]string{}},
		{fmt.Sprintf(baseCompleteResponse, versionLine, "", "", ""),
			testDefaultRelease, testDefaultChannel, testDefaultArch,
			[]string{versionID}},
		{fmt.Sprintf(baseCompleteResponse, versionLine, versionPlusOneLine, "", ""),
			testDefaultRelease, testDefaultChannel, testDefaultArch,
			[]string{versionPlusOneID, versionID}},
		{fmt.Sprintf(baseCompleteResponse, versionPlusOneLine, versionLine, "", ""),
			testDefaultRelease, testDefaultChannel, testDefaultArch,
			[]string{versionPlusOneID, versionID}},
		{fmt.Sprintf(baseCompleteResponse, versionPlusOneLine, versionLine, "", versionPlusTwoLine),
			testDefaultRelease, testDefaultChannel, testDefaultArch,
			[]string{versionPlusTwoID, versionPlusOneID, versionID}},
		{fmt.Sprintf(baseCompleteResponse, versionPlusOneLine, versionPlusTwoLine, versionLine, versionPlusOneLine),
			testDefaultRelease, testDefaultChannel, testDefaultArch,
			[]string{versionPlusTwoID, versionPlusOneID, versionPlusOneID, versionID}},
	}
	for _, item := range testCases {
		s.cli.output = item.glanceOutput
		imageList, _ := s.subject.GetVersions(item.release, item.channel, item.arch)

		c.Check(testEq(imageList, item.expectedImageNames), check.Equals, true)
	}
}

func (s *cloudSuite) TestGetVersionsQueriesGlance(c *check.C) {
	s.subject.GetVersions(testDefaultRelease, testDefaultChannel, testDefaultArch)

	c.Assert(s.cli.execCommandCalls["openstack image list --property status=active"], check.Equals, 1)
}

func getIDFromGlanceResponse(response string) string {
	// response is of the form:
	// | 762d5ce2-fbc2-4685-8d6c-71249d19df9e | ubuntu-core/custom/ubuntu-%s-snappy-core-%s-%s-%d-disk1.img                        |
	items := strings.Fields(response)
	return items[3]
}

func testEq(a, b []string) bool {

	if a == nil && b == nil {
		return true
	}

	if a == nil || b == nil {
		return false
	}

	if len(a) != len(b) {
		return false
	}

	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}

	return true
}
