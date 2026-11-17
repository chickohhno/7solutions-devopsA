resource "google_compute_instance" "ansible_vm" {
  name         = "ansible-vm"
  machine_type = "e2-small"
  zone         = "${var.region}-a"

  tags = ["ansible-ssh"]

  boot_disk {
    initialize_params {
      image = "ubuntu-os-cloud/ubuntu-2404-lts-amd64"
      size  = 20
      type  = "pd-balanced"
    }
  }

  network_interface {
    subnetwork = google_compute_subnetwork.gke_subnet.id

    access_config {}
  }

  metadata = {
    ssh-keys = "ansible:${trimspace(file(pathexpand("~/.ssh/ansible_gce.pub")))}"
  }

  labels = {
    environment = "lab"
    managed_by  = "terraform"
  }
}

resource "google_compute_firewall" "ansible_ssh" {
  name    = "allow-ansible-ssh"
  network = google_compute_network.vpc.name

  allow {
    protocol = "tcp"
    ports    = ["22"]
  }

  source_ranges = [var.admin_cidr]
  target_tags   = ["ansible-ssh"]
}

output "ansible_vm_public_ip" {
  value = google_compute_instance.ansible_vm.network_interface[0].access_config[0].nat_ip
}