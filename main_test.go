package main

import "testing"

// TODO:
// how to do cleanup and setup for tests
// how to create temp dir using testing package

func TestShouldSkipWhenMediaExists(t *testing.T) {
	t.Error("testing error")
}

func TestShouldMoveWhenMediaDoesNotExists(t *testing.T) {
}

func TestShouldMoveToSoocDirWhenProcessingJpgMedia(t *testing.T) {
}

func TestShouldMoveToPhotosDirWhenProcessingJpgOrRafMedia(t *testing.T) {
}

func TestShouldMoveToVideosDirWhenProcessingMovMedia(t *testing.T) {
}

func TestShouldSkipWhenMediaIsCopyButNameIsDifferent(t *testing.T) {
}

func TestShouldMoveWhenMediaExistsButIsNotPlacedCorrectly(t *testing.T) {
}

func TestShouldErrorWhenMetadataNotPresent(t *testing.T) {
}

// TODO: test dynamic hash chunk using mocking
