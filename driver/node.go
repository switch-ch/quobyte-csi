package driver

import (
	"context"
	"fmt"
	"os"
	"strings"

	csi "github.com/container-storage-interface/spec/lib/go/csi/v0"
	"github.com/golang/glog"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// NodePublishVolume mounts the volume to the pod with the given target path
// QuobyteClient does the mounting of the volumes
func (d *QuobyteDriver) NodePublishVolume(ctx context.Context, req *csi.NodePublishVolumeRequest) (*csi.NodePublishVolumeResponse, error) {
	targetPath := req.GetTargetPath()
	readonly := req.Readonly
	volumeId := req.GetVolumeId()
	// Incase of preprovisioned volumes, NodePublishSecrets are not taken from storage class but
	// needs to be passed as nodePublishSecretRef in PV (kubernetes) definition
	secrets := req.GetNodePublishSecrets()

	volParts := strings.Split(volumeId, "|")

	if len(volParts) != 3 {
		return nil, fmt.Errorf("given volumeHandle '%s' is not in the format <API_URL>|<TENANT_NAME/TENANT_UUID>|<VOL_NAME/VOL_UUID>", volumeId)
	}
	if len(targetPath) == 0 {
		return nil, fmt.Errorf("given target mount path is empty")
	}

	var volUUID string

	if len(secrets) == 0 {
		glog.Infof("csiNodePublishSecret is  not recieved. Assuming volume given with UUID")
		volUUID = volParts[2]
	} else {
		quobyteClient, err := getAPIClient(secrets, volParts[0])
		if err != nil {
			return nil, err
		}
		// volume name should be retrieved from the req.GetVolumeId()
		// Due to csi lacking in parameter passing during delete Volume, req.volumeId is changed
		// to <API_URL>|<TENANT_NAME/TENANT_UUID>|<VOL_NAME/VOL_UUID>. see controller.go CreateVolume for the details.

		volUUID, err = quobyteClient.GetVolumeUUID(volParts[2], volParts[1])
		if err != nil {
			return nil, err
		}
	}

	var options []string
	if readonly {
		options = append(options, "ro")
	}
	volCap := req.GetVolumeCapability()
	if volCap != nil {
		mount := volCap.GetMount()
		if mount != nil {
			mntFlags := mount.GetMountFlags()
			if mntFlags != nil {
				options = append(options, mntFlags...)
			}
		}
	}

	err := Mount(fmt.Sprintf("%s/%s", d.clientMountPoint, volUUID), targetPath, "quobyte", options)
	if err != nil {
		return nil, err
	}

	return &csi.NodePublishVolumeResponse{}, nil
}

// NodeUnpublishVolume Currently not implemented as Quobyte has only single mount point
func (d *QuobyteDriver) NodeUnpublishVolume(ctx context.Context, req *csi.NodeUnpublishVolumeRequest) (*csi.NodeUnpublishVolumeResponse, error) {
	target := req.GetTargetPath()
	if len(target) == 0 {
		return nil, fmt.Errorf("target for unmount is empty")
	}
	glog.Info("Unmounting %s", target)
	err := Unmount(target)
	if err != nil {
		return nil, err
	}
	return &csi.NodeUnpublishVolumeResponse{}, nil
}

// NodeGetId returns the unique node ID, currently unique id is hostname of the node
func (d *QuobyteDriver) NodeGetId(ctx context.Context, req *csi.NodeGetIdRequest) (*csi.NodeGetIdResponse, error) {
	host, err := os.Hostname()
	if err != nil {
		return nil, err
	}
	return &csi.NodeGetIdResponse{
		NodeId: host,
	}, nil
}

// NodeGetCapabilities returns the capabilities of the node server
func (d *QuobyteDriver) NodeGetCapabilities(ctx context.Context, req *csi.NodeGetCapabilitiesRequest) (*csi.NodeGetCapabilitiesResponse, error) {
	return &csi.NodeGetCapabilitiesResponse{
		Capabilities: []*csi.NodeServiceCapability{
			{
				Type: &csi.NodeServiceCapability_Rpc{
					Rpc: &csi.NodeServiceCapability_RPC{
						Type: csi.NodeServiceCapability_RPC_UNKNOWN,
					},
				},
			},
		},
	}, nil
}

// NodeStageVolume Stages the volume to the node under /mnt/quobyte
func (d *QuobyteDriver) NodeStageVolume(ctx context.Context, req *csi.NodeStageVolumeRequest) (*csi.NodeStageVolumeResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "NodeStageVolume: Not implented by Quobyte CSI")
}

// NodeUnstageVolume Unstages the volume from /mnt/quobyte
func (d *QuobyteDriver) NodeUnstageVolume(ctx context.Context, req *csi.NodeUnstageVolumeRequest) (*csi.NodeUnstageVolumeResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "NodeUnstageVolume: Not implented by Quobyte CSI")
}
