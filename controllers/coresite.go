package controllers

var coreSite = `<?xml version="1.0" encoding="UTF-8"?>
<?xml-stylesheet type="text/xsl" href="configuration.xsl"?>
<!--
  Licensed under the Apache License, Version 2.0 (the "License");
  you may not use this file except in compliance with the License.
  You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

  Unless required by applicable law or agreed to in writing, software
  distributed under the License is distributed on an "AS IS" BASIS,
  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
  See the License for the specific language governing permissions and
  limitations under the License. See accompanying LICENSE file.
-->

<!-- Put site-specific property overrides in this file. -->

<configuration>
<property><name>hadoop.proxyuser.hue.hosts</name><value>*</value></property>
<property><name>fs.defaultFS</name><value>hdfs://namenode:8020</value></property>
<!-- <property><name>hadoop.proxyuser.hue.groups</name><value>*</value></property> -->
<property><name>hadoop.http.staticuser.user</name><value>root</value></property>
<property><name>hadoop.proxyuser.hue.hosts</name><value>*</value></property>
<property><name>hadoop.proxyuser.hue.groups</name><value>*</value></property>
<property><name>hadoop.http.staticuser.user</name><value>root</value></property>
<property>
		<name>fs.defaultFS</name>
		<value>cosn://%s</value>
	</property>

	<property>
		<name>fs.cosn.credentials.provider</name>
		<value>org.apache.hadoop.fs.auth.SimpleCredentialProvider</value>
		<description>

			This option allows the user to specify how to get the credentials.
			Comma-separated class names of credential provider classes which implement
			com.qcloud.cos.auth.COSCredentialsProvider:

			1.org.apache.hadoop.fs.auth.SessionCredentialProvider: Obtain the secretId and secretKey from the URI:cosn://secretId:secretKey@example-1250000000000/;
			2.org.apache.hadoop.fs.auth.SimpleCredentialProvider: Obtain the secret id and secret key
			from fs.cosn.userinfo.secretId and fs.cosn.userinfo.secretKey in core-site.xml;
			3.org.apache.hadoop.fs.auth.EnvironmentVariableCredentialProvider: Obtain the secret id and secret key
			from system environment variables named COS_SECRET_ID and COS_SECRET_KEY.

			If unspecified, the default order of credential providers is:
			1. org.apache.hadoop.fs.auth.SessionCredentialProvider
			2. org.apache.hadoop.fs.auth.SimpleCredentialProvider
			3. org.apache.hadoop.fs.auth.EnvironmentVariableCredentialProvider
			4. org.apache.hadoop.fs.auth.CVMInstanceCredentialsProvider
			5. org.apache.hadoop.fs.auth.CPMInstanceCredentialsProvider
		</description>
	</property>

	<property>
		<name>fs.cosn.userinfo.secretId</name>
		<value>%s</value>
		<description>Tencent Cloud Secret Id</description>
	</property>

	<property>
		<name>fs.cosn.userinfo.secretKey</name>
		<value>%s</value>
		<description>Tencent Cloud Secret Key</description>
	</property>

	<property>
		<name>fs.cosn.bucket.region</name>
		<value>%s</value>
		<description>The region where the bucket is located</description>
	</property>

	<property>
		<name>fs.cosn.impl</name>
		<value>org.apache.hadoop.fs.CosFileSystem</value>
		<description>The implementation class of the CosN Filesystem</description>
	</property>

	<property>
		<name>fs.AbstractFileSystem.cosn.impl</name>
		<value>org.apache.hadoop.fs.CosN</value>
		<description>The implementation class of the CosN AbstractFileSystem.</description>
	</property>

	<property>
		<name>fs.cosn.tmp.dir</name>
		<value>/tmp/hadoop_cos</value>
		<description>Temporary files will be placed here.</description>
	</property>

	<property>
		<name>fs.cosn.upload.buffer</name>
		<value>mapped_disk</value>
		<description>The type of upload buffer. Available values: non_direct_memory, direct_memory, mapped_disk</description>
	</property>

	<property>
		<name>fs.cosn.upload.buffer.size</name>
		<value>33554432</value>
		<description>The total size of the upload buffer pool. -1 means unlimited.</description>
	</property>

	<property>
		<name>fs.cosn.upload.part.size</name>
		<value>8388608</value>
		<description>The part size for MultipartUpload.
		Considering the COS supports up to 10000 blocks, user should estimate the maximum size of a single file.
		For example, 8MB part size can allow  writing a 78GB single file.</description>
	</property>
	<property>
		<name>fs.cosn.maxRetries</name>
		<value>200</value>
		<description>The maximum number of retries for reading or writing files to
	COS, before we signal failure to the application.</description>
	</property>

	<property>
		<name>fs.cosn.retry.interval.seconds</name>
		<value>3</value>
		<description>The number of seconds to sleep between each COS retry.</description>
	</property>

	<property>
		<name>fs.cosn.read.ahead.block.size</name>
		<value>1048576</value>
		<description>
			Bytes to read ahead during a seek() before closing and
			re-opening the cosn HTTP connection.
		</description>
	</property>

	<property>
		<name>fs.cosn.read.ahead.queue.size</name>
		<value>8</value>
		<description>The length of the pre-read queue.</description>
	</property>

	<property>
		<name>fs.cosn.customer.domain</name>
		<value></value>
		<description>The customer domain.</description>
	</property>

	<property>
		<name>fs.cosn.server-side-encryption.algorithm</name>
		<value></value>
		<description>The server side encryption algorithm.</description>
	</property>

	 <property>
		<name>fs.cosn.server-side-encryption.key</name>
		<value></value>
		<description>The SSE-C server side encryption key.</description>
	</property>
<property><name>hadoop.proxyuser.hue.hosts</name><value>*</value></property>
<property><name>fs.defaultFS</name><value>hdfs://namenode:8020</value></property>
<property><name>hadoop.proxyuser.hue.groups</name><value>*</value></property>
<property><name>hadoop.http.staticuser.user</name><value>root</value></property>
<property><name>hadoop.proxyuser.hue.hosts</name><value>*</value></property>
<property><name>fs.defaultFS</name><value>hdfs://namenode:8020</value></property>
<property><name>hadoop.proxyuser.hue.groups</name><value>*</value></property>
<property><name>hadoop.http.staticuser.user</name><value>root</value></property>
</configuration>`
