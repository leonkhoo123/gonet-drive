import axiosLayer from './axiosLayer';   // axios instance WITHOUT token
import { generateOpId } from "../utils/id";


export const postDisqualified = async (filePath: string, opId: string = generateOpId()) => {
  const rs = await axiosLayer.post(
    "/user/video/disqualified",
    { path: filePath, opId }, // request body
    {
      headers: { "Content-Type": "application/json" },
    }
  );
  return rs.data;
};

export const renameFileMoveToDone = async (filePath: string, name: string, angle: number, opId: string = generateOpId()) => {

  const rs = await axiosLayer.post(
    "/user/video/rename-done",
    {
      path: filePath,
      newName: name,
      rotateAngle: angle,
      opId
    },
    { headers: { "Content-Type": "application/json" } }
  );
  return rs.data;
};

