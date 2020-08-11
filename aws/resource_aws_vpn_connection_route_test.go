package aws

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/ec2"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccAWSVpnConnectionRoute_basic(t *testing.T) {
	rBgpAsn := acctest.RandIntRange(64512, 65534)
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccAwsVpnConnectionRouteDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAwsVpnConnectionRouteConfig(rBgpAsn),
				Check: resource.ComposeTestCheckFunc(
					testAccAwsVpnConnectionRoute("aws_vpn_connection_route.foo"),
				),
			},
			{
				Config: testAccAwsVpnConnectionRouteConfigUpdate(rBgpAsn),
				Check: resource.ComposeTestCheckFunc(
					testAccAwsVpnConnectionRoute("aws_vpn_connection_route.foo"),
				),
			},
		},
	})
}

func testAccAwsVpnConnectionRouteDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).ec2conn
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_vpn_connection_route" {
			continue
		}

		cidrBlock, vpnConnectionId := resourceAwsVpnConnectionRouteParseId(rs.Primary.ID)

		routeFilters := []*ec2.Filter{
			{
				Name:   aws.String("route.destination-cidr-block"),
				Values: []*string{aws.String(cidrBlock)},
			},
			{
				Name:   aws.String("vpn-connection-id"),
				Values: []*string{aws.String(vpnConnectionId)},
			},
		}

		resp, err := conn.DescribeVpnConnections(&ec2.DescribeVpnConnectionsInput{
			Filters: routeFilters,
		})
		if err != nil {
			if ec2err, ok := err.(awserr.Error); ok && ec2err.Code() == "InvalidVpnConnectionID.NotFound" {
				// not found, all good
				return nil
			}
			return err
		}

		var vpnc *ec2.VpnConnection
		if resp != nil {
			// range over the connections and isolate the one we created
			for _, v := range resp.VpnConnections {
				if *v.VpnConnectionId == vpnConnectionId {
					vpnc = v
				}
			}

			if vpnc == nil {
				// vpn connection not found, so that's good...
				return nil
			}

			if vpnc.State != nil && *vpnc.State == "deleted" {
				return nil
			}
		}

	}
	return fmt.Errorf("Fall through error, Check Destroy criteria not met")
}

func testAccAwsVpnConnectionRoute(vpnConnectionRouteResource string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[vpnConnectionRouteResource]
		if !ok {
			return fmt.Errorf("Not found: %s", vpnConnectionRouteResource)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}
		route, ok := s.RootModule().Resources[vpnConnectionRouteResource]
		if !ok {
			return fmt.Errorf("Not found: %s", vpnConnectionRouteResource)
		}

		cidrBlock, vpnConnectionId := resourceAwsVpnConnectionRouteParseId(route.Primary.ID)

		routeFilters := []*ec2.Filter{
			{
				Name:   aws.String("route.destination-cidr-block"),
				Values: []*string{aws.String(cidrBlock)},
			},
			{
				Name:   aws.String("vpn-connection-id"),
				Values: []*string{aws.String(vpnConnectionId)},
			},
		}

		ec2conn := testAccProvider.Meta().(*AWSClient).ec2conn

		_, err := ec2conn.DescribeVpnConnections(&ec2.DescribeVpnConnectionsInput{
			Filters: routeFilters,
		})
		return err
	}
}

func testAccAwsVpnConnectionRouteConfig(rBgpAsn int) string {
	return fmt.Sprintf(`
resource "aws_vpn_gateway" "vpn_gateway" {
  tags = {
    Name = "vpn_gateway"
  }
}

resource "aws_customer_gateway" "customer_gateway" {
  bgp_asn    = %d
  ip_address = "182.0.0.1"
  type       = "ipsec.1"
}

resource "aws_vpn_connection" "vpn_connection" {
  vpn_gateway_id      = "${aws_vpn_gateway.vpn_gateway.id}"
  customer_gateway_id = "${aws_customer_gateway.customer_gateway.id}"
  type                = "ipsec.1"
  static_routes_only  = true
}

resource "aws_vpn_connection_route" "foo" {
  destination_cidr_block = "172.168.10.0/24"
  vpn_connection_id      = "${aws_vpn_connection.vpn_connection.id}"
}
`, rBgpAsn)
}

// Change destination_cidr_block
func testAccAwsVpnConnectionRouteConfigUpdate(rBgpAsn int) string {
	return fmt.Sprintf(`
resource "aws_vpn_gateway" "vpn_gateway" {
  tags = {
    Name = "vpn_gateway"
  }
}

resource "aws_customer_gateway" "customer_gateway" {
  bgp_asn    = %d
  ip_address = "182.0.0.1"
  type       = "ipsec.1"
}

resource "aws_vpn_connection" "vpn_connection" {
  vpn_gateway_id      = "${aws_vpn_gateway.vpn_gateway.id}"
  customer_gateway_id = "${aws_customer_gateway.customer_gateway.id}"
  type                = "ipsec.1"
  static_routes_only  = true
}

resource "aws_vpn_connection_route" "foo" {
  destination_cidr_block = "172.168.20.0/24"
  vpn_connection_id      = "${aws_vpn_connection.vpn_connection.id}"
}
`, rBgpAsn)
}
