package handlers

import (
	"io"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/masoniclounge/masoniccore/internal/services"
	"github.com/masoniclounge/masoniccore/internal/storage"
)

// Media exposes upload and download endpoints backed by S3.
type Media struct {
	svc *services.MediaService
	s3  *storage.S3
}

// NewMedia creates a Media handler.
func NewMedia(svc *services.MediaService, s3 *storage.S3) *Media {
	return &Media{svc: svc, s3: s3}
}

// UploadAvatar stores the uploaded image as the caller's avatar.
func (h *Media) UploadAvatar(w http.ResponseWriter, r *http.Request) {
	user := authUser(r)
	if user == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "missing_file", "a multipart file field 'file' is required")
		return
	}
	defer file.Close()

	u, err := h.svc.UploadAvatar(r.Context(), user.ID, header.Filename, header.Header.Get("Content-Type"), header.Size, file)
	if err != nil {
		writeCoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, u)
}

// UploadAttachment stores an uploaded file, optionally tied to a post.
func (h *Media) UploadAttachment(w http.ResponseWriter, r *http.Request) {
	user := authUser(r)
	if user == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "missing_file", "a multipart file field 'file' is required")
		return
	}
	defer file.Close()

	var postID *uuid.UUID
	if raw := r.FormValue("post_id"); raw != "" {
		id, err := uuid.Parse(raw)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_id", "Invalid post id")
			return
		}
		postID = &id
	}

	a, err := h.svc.UploadAttachment(r.Context(), user.ID, postID, header.Filename, header.Header.Get("Content-Type"), header.Size, file)
	if err != nil {
		writeCoreError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, a)
}

// DeleteAttachment removes an attachment owned by the caller.
func (h *Media) DeleteAttachment(w http.ResponseWriter, r *http.Request) {
	user := authUser(r)
	if user == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}
	id, ok := pathID(r, "attachmentID")
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid_id", "Invalid attachment id")
		return
	}
	if err := h.svc.DeleteAttachment(r.Context(), id, user.ID, user.Roles); err != nil {
		writeCoreError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// Serve streams an object from S3 at /media/{bucket}/*.
func (h *Media) Serve(w http.ResponseWriter, r *http.Request) {
	if chi.URLParam(r, "bucket") != h.s3.Bucket() {
		writeError(w, http.StatusNotFound, "media_not_found", "File not found")
		return
	}
	key := chi.URLParam(r, "*")
	if key == "" || key[0] == '/' {
		writeError(w, http.StatusNotFound, "media_not_found", "File not found")
		return
	}

	obj, err := h.s3.Open(r.Context(), key)
	if err != nil {
		writeError(w, http.StatusNotFound, "media_not_found", "File not found")
		return
	}
	defer obj.Close()

	stat, err := obj.Stat()
	if err != nil {
		writeError(w, http.StatusNotFound, "media_not_found", "File not found")
		return
	}

	w.Header().Set("Content-Type", stat.ContentType)
	w.Header().Set("Content-Length", strconv.FormatInt(stat.Size, 10))
	w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	if _, err := io.Copy(w, obj); err != nil {
		return
	}
}
