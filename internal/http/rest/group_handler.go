package rest

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/bwise1/waze_kibris/internal/model"
	"github.com/bwise1/waze_kibris/util"
	"github.com/bwise1/waze_kibris/util/tracing"
	"github.com/bwise1/waze_kibris/util/values"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

func (api *API) GroupRoutes() chi.Router {
	mux := chi.NewRouter()

	mux.Group(func(r chi.Router) {
		r.Use(api.RequireLogin)

		r.Method(http.MethodPost, "/", Handler(api.CreateCommunityGroupHandler))
		//(e.g., public groups, groups nearby, user's groups)
		// Query Params: ?query=..., ?nearby=lat,lon,radius, ?member=me, ?public=true/false, ?page=1, ?pageSize=20
		// Response: List of groups matching criteria
		r.Method(http.MethodGet, "/", Handler(api.SearchForListOfGroupsHandler))
		// Get details of a specific group
		// Response: Full group details (incl. member count, maybe recent messages preview)
		r.Method(http.MethodGet, "/{groupID}", Handler(api.GetGroupByIDHandler))
		// Update group details (Name, Description, Icon, Privacy) - Requires Admin role
		// Request Body: { "name": "...", "description": "...", "is_private": bool, "icon_url": "..." }
		r.Method(http.MethodPut, "/{groupID}", Handler(api.placeHolderHandler))
		// Delete a group - Requires Admin/Owner role
		// Response: Success/Failure message
		r.Method(http.MethodDelete, "/{groupID}", Handler(api.placeHolderHandler))
		// Join a public group / Request to join a private group
		// Response: Membership details or Pending status
		r.Method(http.MethodPost, "/{short_code}/join", Handler(api.JoinGroupByShortCodeHandler)) // Or POST to /{groupID}/members using authenticated user ID
		// Leave a group
		// Response: Success/Failure message
		r.Method(http.MethodDelete, "/{groupID}/leave", Handler(api.LeaveGroupHandler)) // Or DELETE /{groupID}/members/me
		// List members of a group
		// Query Params: ?page=1, ?pageSize=50
		// Response: List of members (User ID, Username, Role)
		r.Method(http.MethodGet, "/{groupID}/members", Handler(api.placeHolderHandler))
		// Manage group members (Admin actions)
		// Update a member's role (e.g., promote to admin) - Requires Admin role
		// Request Body: { "role": "admin/member" }
		// Response: Updated membership details
		r.Method(http.MethodPut, "/{groupID}/members/{userID}", Handler(api.placeHolderHandler))
		// Remove (kick) a member from a group - Requires Admin role
		// Response: Success/Failure message
		r.Method(http.MethodDelete, "/{groupID}/members/{userID}", Handler(api.placeHolderHandler))
		// Invite a user to the group - Requires Admin/Member (configurable)
		// Request Body: { "user_id": "..." }
		// Response: Invitation details or Success/Failure
		r.Method(http.MethodPost, "/{groupID}/invitations", Handler(api.placeHolderHandler))
		// List pending invitations for the group - Requires Admin role
		// Response: List of pending invitations
		r.Method(http.MethodGet, "/{groupID}/invitations", Handler(api.placeHolderHandler))
		// User actions on invitations (could be top-level or user-scoped)
		// Accept an invitation
		// Response: Success/Failure (results in membership creation)
		r.Method(http.MethodPost, "/invitations/{invitationID}/accept", Handler(api.placeHolderHandler)) // Assumes a top-level /invitations route exists
		// Decline an invitation
		// Response: Success/Failure
		r.Method(http.MethodPost, "/invitations/{invitationID}/decline", Handler(api.placeHolderHandler)) // Assumes a top-level /invitations route exists
		// List user's pending invitations
		// Response: List of invitations for the logged-in user
		r.Method(http.MethodGet, "/users/me/invitations", Handler(api.placeHolderHandler)) // Or similar user-scoped route
		// Send a message to the group - Requires Member role
		// Request Body: { "content": "...", "message_type": "text/location/report_link", "attachment_url": "..." }
		// Response: The created message details
		r.Method(http.MethodPost, "/{groupID}/messages", Handler(api.SendGroupMessageHandler))

		// Get messages from the group - Requires Member role
		// Query Params: ?before=<messageID/timestamp>, ?after=<messageID/timestamp>, ?limit=50
		// Response: List of messages (paginated)
		r.Method(http.MethodGet, "/{groupID}/messages", Handler(api.GetGroupMessagesHandler))

		// (Optional) Update a message - Requires Author role (within time limit?)
		// Request Body: { "content": "..." }
		// Response: Updated message details
		r.Method(http.MethodPut, "/{groupID}/messages/{messageID}", Handler(api.placeHolderHandler))

		// (Optional) Delete a message - Requires Author or Admin role
		// Response: Success/Failure message
		r.Method(http.MethodDelete, "/{groupID}/messages/{messageID}", Handler(api.placeHolderHandler))

		// (Optional) Mark messages as read - Requires Member role
		// Request Body: { "last_read_message_id": "..." } or { "last_read_timestamp": "..." }
		// Response: Success/Failure message
		r.Method(http.MethodPost, "/{groupID}/read", Handler(api.placeHolderHandler))

	})

	return mux
}

func (api *API) placeHolderHandler(_ http.ResponseWriter, r *http.Request) *ServerResponse {
	return &ServerResponse{
		Message:    "Not yet Implemented",
		Status:     values.Success,
		StatusCode: util.StatusCode(values.Success),
	}
}

func (api *API) GetGroupMessagesHandler(w http.ResponseWriter, r *http.Request) *ServerResponse {
	tc := r.Context().Value(values.ContextTracingKey).(tracing.Context)
	groupIDStr := chi.URLParam(r, "groupID")
	groupID, err := uuid.Parse(groupIDStr)
	if err != nil {
		return respondWithError(err, "invalid group ID format", values.BadRequestBody, &tc)
	}

	messages, err := api.GetGroupMessages(r.Context(), groupID, 50)
	if err != nil {
		return respondWithError(err, "failed to get group messages", values.Failed, &tc)
	}

	return &ServerResponse{
		Message:    "Messages retrieved",
		Status:     values.Success,
		StatusCode: util.StatusCode(values.Success),
		Data:       messages,
	}
}

func (api *API) LeaveGroupHandler(w http.ResponseWriter, r *http.Request) *ServerResponse {
	tc := r.Context().Value(values.ContextTracingKey).(tracing.Context)
	groupIDStr := chi.URLParam(r, "groupID")
	groupID, err := uuid.Parse(groupIDStr)
	if err != nil {
		return respondWithError(err, "invalid group ID format", values.BadRequestBody, &tc)
	}

	userID, err := util.GetUserIDFromContext(r.Context())
	if err != nil {
		return respondWithError(err, "unable to get user ID from context", values.NotAuthorised, &tc)
	}

	err = api.LeaveCommunityGroup(r.Context(), groupID, userID)
	if err != nil {
		return respondWithError(err, "failed to leave group", values.Failed, &tc)
	}

	return &ServerResponse{
		Message:    "Successfully left the group",
		Status:     values.Success,
		StatusCode: util.StatusCode(values.Success),
	}
}

func (api *API) SendGroupMessageHandler(w http.ResponseWriter, r *http.Request) *ServerResponse {
	tc := r.Context().Value(values.ContextTracingKey).(tracing.Context)
	groupIDStr := chi.URLParam(r, "groupID")
	groupID, err := uuid.Parse(groupIDStr)
	if err != nil {
		return respondWithError(err, "invalid group ID format", values.BadRequestBody, &tc)
	}

	userID, err := util.GetUserIDFromContext(r.Context())
	if err != nil {
		return respondWithError(err, "unable to get user ID from context", values.NotAuthorised, &tc)
	}

	var req model.GroupMessage
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return respondWithError(err, "Invalid request payload", values.BadRequestBody, &tc)
	}

	req.GroupID = groupID
	req.UserID = userID
	if req.MessageType == "" {
		req.MessageType = "text"
	}

	savedMsg, err := api.InsertGroupMessage(r.Context(), req)
	if err != nil {
		return respondWithError(err, "Failed to send message", values.Failed, &tc)
	}

	// Broadcast the message via WebSockets
	wsMsg, _ := json.Marshal(savedMsg)
	api.Deps.WebSocket.BroadcastToGroup(groupID.String(), wsMsg)

	return &ServerResponse{
		Message:    "Message sent successfully",
		Status:     values.Success,
		StatusCode: util.StatusCode(values.Success),
		Data:       savedMsg,
	}
}

func (api *API) CreateCommunityGroupHandler(_ http.ResponseWriter, r *http.Request) *ServerResponse {
	tc := r.Context().Value(values.ContextTracingKey).(tracing.Context)
	var req model.CommunityGroup
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return respondWithError(err, "Invalid request payload", values.BadRequestBody, &tc)
	}
	userId, err := util.GetUserIDFromContext(r.Context())
	if err != nil {
		return respondWithError(err, "unable to get user ID from context", values.NotAuthorised, &tc)
	}

	req.CreatorID = userId
	group, status, message, err := api.CreateGroupHelper(r.Context(), req)
	if err != nil {
		return respondWithError(err, "Failed to create group", values.Failed, &tc)
	}

	return &ServerResponse{
		Message:    message,
		Status:     status,
		StatusCode: util.StatusCode(status),
		Data:       group,
	}
}

func (api *API) SearchForListOfGroupsHandler(_ http.ResponseWriter, r *http.Request) *ServerResponse {
	tc := r.Context().Value(values.ContextTracingKey).(tracing.Context)

	log.Println("here in handler")
	groups, status, message, err := api.SearchCommunityGroupsHelper(r.Context())
	if err != nil {
		return respondWithError(err, "unable to get groups", values.Failed, &tc)
	}
	return &ServerResponse{
		Message:    message,
		Status:     status,
		StatusCode: util.StatusCode(status),
		Data:       groups,
	}
}

func (api *API) JoinGroupByShortCodeHandler(w http.ResponseWriter, r *http.Request) *ServerResponse {
	tc := r.Context().Value(values.ContextTracingKey).(tracing.Context)
	shortCode := chi.URLParam(r, "short_code")
	userID, err := util.GetUserIDFromContext(r.Context())
	if err != nil {
		return respondWithError(err, "unable to get user ID from context", values.NotAuthorised, &tc)
	}

	// Find the group by short_code
	group, err := api.GetCommunityGroupByShortCode(r.Context(), shortCode)
	if err != nil {
		return respondWithError(err, "Group not found", values.Failed, &tc)
	}

	// Determine membership status
	status := "active"
	if group.Visibility == "private" {
		status = "pending"
	}

	// Insert membership (handle duplicate gracefully)
	_, err = api.Deps.DB.Pool().Exec(r.Context(), `
        INSERT INTO group_memberships (group_id, user_id, role, status, joined_at, updated_at)
        VALUES ($1, $2, 'member', $3, NOW(), NOW())
        ON CONFLICT (group_id, user_id) DO NOTHING
    `, group.ID, userID, status)
	if err != nil {
		return respondWithError(err, "Failed to join group", values.Failed, &tc)
	}

	message := "Joined group successfully"
	if status == "pending" {
		message = "Join request sent, awaiting admin approval"
	}

	return &ServerResponse{
		Message:    message,
		Status:     values.Success,
		StatusCode: util.StatusCode(values.Success),
	}
}

func (api *API) GetGroupByIDHandler(w http.ResponseWriter, r *http.Request) *ServerResponse {
	tc := r.Context().Value(values.ContextTracingKey).(tracing.Context)
	groupIDStr := chi.URLParam(r, "groupID")
	groupID, err := uuid.Parse(groupIDStr)
	if err != nil {
		return respondWithError(err, "invalid group ID format", values.BadRequestBody, &tc)
	}

	group, err := api.GetCommunityGroupByID(r.Context(), groupID)
	if err != nil {
		return respondWithError(err, "Group not found", values.Failed, &tc)
	}

	return &ServerResponse{
		Message:    "Group details retrieved",
		Status:     values.Success,
		StatusCode: util.StatusCode(values.Success),
		Data:       group,
	}
}
