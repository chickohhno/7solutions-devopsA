output "gke_cluster_name" {
  value = google_container_cluster.hello_api.name
}

output "gke_cluster_region" {
  value = google_container_cluster.hello_api.location
}

output "gke_node_service_account" {
  value = google_service_account.gke_nodes.email
}