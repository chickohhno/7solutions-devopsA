resource "google_service_account" "gke_nodes" {
  account_id   = "hello-api-gke-nodes"
  display_name = "Hello API GKE Node Service Account"
}

resource "google_project_iam_member" "gke_nodes" {
  project = var.projec_id
  role    = "roles/container.defaultNodeServiceAccount"
  member  = "serviceAccount:${google_service_account.gke_nodes.email}"
}