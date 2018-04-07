data "aws_route53_zone" "main" {
  count = "${var.route53_enabled == "true" ? 1 : 0}"

  zone_id = "${var.route53_zone_id}"
}

resource "aws_route53_record" "elb" {
  count = "${var.route53_enabled == "true" ? 1 : 0}"

  zone_id = "${data.aws_route53_zone.main.zone_id}"
  name    = "etcd.${data.aws_route53_zone.main.name}"
  type    = "A"

  alias {
    name                   = "${aws_elb.clients.dns_name}"
    zone_id                = "${aws_elb.clients.zone_id}"
    evaluate_target_health = true
  }
}