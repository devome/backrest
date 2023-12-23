import React from "react";
import { Repo } from "../../gen/ts/v1/config_pb";
import { Flex, Tabs, Typography } from "antd";
import { useRecoilValue } from "recoil";
import { configState } from "../state/config";
import { useAlertApi } from "../components/Alerts";
import { OperationList } from "../components/OperationList";
import { OperationTree } from "../components/OperationTree";
import { MAX_OPERATION_HISTORY } from "../constants";
import { GetOperationsRequest } from "../../gen/ts/v1/service_pb";

export const RepoView = ({ repo }: React.PropsWithChildren<{ repo: Repo }>) => {
  const alertsApi = useAlertApi()!;

  // Gracefully handle deletions by checking if the plan is still in the config.
  const config = useRecoilValue(configState);
  let repoInConfig = config.repos?.find((p) => p.id === repo.id);
  if (!repoInConfig) {
    return <p>Repo was deleted.</p>;
  }
  repo = repoInConfig;

  return (
    <>
      <Flex gap="small" align="center" wrap="wrap">
        <Typography.Title>
          {repo.id}
        </Typography.Title>
      </Flex>
      <Tabs
        defaultActiveKey="1"
        items={[
          {
            key: "1",
            label: "Tree View",
            children: (
              <>
                <OperationTree
                  req={new GetOperationsRequest({ repoId: repo.id!, lastN: BigInt(MAX_OPERATION_HISTORY) })}
                />
              </>
            ),
          },
          {
            key: "2",
            label: "Operation List",
            children: (
              <>
                <h2>Backup Action History</h2>
                <OperationList
                  req={new GetOperationsRequest({ repoId: repo.id!, lastN: BigInt(MAX_OPERATION_HISTORY) })}
                  showPlan={true}
                />
              </>
            ),
          },
        ]}
      />
    </>
  );
};