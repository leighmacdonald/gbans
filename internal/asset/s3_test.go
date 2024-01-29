package asset_test

//
//func TestS3Client(t *testing.T) {
//	client, errClient := asset.NewS3Client(
//		zap.NewNop(),
//		"localhost:9000",
//		"gbans-test-key",
//		"gbansgbansgbansgbans",
//		false,
//		"us-east-1")
//
//	testBucket := fmt.Sprintf("test-gbans-%d", time.Now().Unix())
//
//	if errClient != nil {
//		t.Skipf("Cannot initialize client, skipping tests.")
//	}
//
//	if err := client.CreateBucketIfNotExists(context.Background(), testBucket); err != nil {
//		t.Skipf("No server available")
//	}
//
//	randID, _ := uuid.NewV4()
//
//	testFile, errOpen := os.Open("../../testdata/gopher.webp")
//	require.NoError(t, errOpen, "Failed to open test image")
//
//	name, mimeType, size, errGen := asset.GenerateFileMeta(testFile, randID.String())
//	require.NoError(t, errGen)
//
//	_, errSeek := testFile.Seek(0, 0)
//
//	require.NoError(t, errSeek)
//	require.NoError(t, client.Put(context.Background(), testBucket, name, testFile, size, mimeType))
//
//	cli := http.Client{}
//	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, client.LinkObject(testBucket, name), nil)
//	resp, errDl := cli.Do(req)
//
//	require.NoError(t, errDl)
//
//	downloaded, errRead := io.ReadAll(resp.Body)
//	require.NoError(t, errRead)
//
//	require.NoError(t, resp.Body.Close())
//
//	_, _ = testFile.Seek(0, 0)
//
//	expected, errReadFile := io.ReadAll(testFile)
//
//	require.NoError(t, errReadFile)
//
//	require.Equal(t, expected, downloaded)
//	require.NoError(t, client.Remove(context.Background(), testBucket, name))
//	require.NoError(t, client.RemoveBucket(context.Background(), testBucket))
//}
