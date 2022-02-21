# Add-on for Tetra3D > Blender exporting

import bpy, os
from bpy.app.handlers import persistent

bl_info = {
    "name" : "Tetra3D Addon",                        # The name in the addon search menu
    "author" : "SolarLune",
    "description" : "An addon for Tetra3D + Blender",
    "blender" : (3, 0, 1),                             # Lowest version to use
    "location" : "View3D",
    "category" : "Gamedev",
    "version" : (0, 1),
}

boundsTypes = [
    ("NONE", "No Bounds", "No collision will be created for this object.", 0, 0),
    ("AABB", "AABB", "An AABB (axis-aligned bounding box). If the size isn't customized, it will be big enough to fully contain the mesh of the current object. Currently buggy when resolving intersections between AABB or other Triangle Nodes", 0, 1),
    ("CAPSULE", "Capsule", "A capsule, which can rotate. If the radius and height are not set, it will have a radius and height to fully contain the current object", 0, 2),
    ("SPHERE", "Sphere", "A sphere. If the radius is not custom set, it will have a large enough radius to fully contain the provided object", 0, 3),
    ("TRIANGLES", "Triangle Mesh", "A triangle mesh bounds type. Only works on mesh-type objects (i.e. an Empty won't generate a BoundingTriangles). Accurate, but slow. Currently buggy when resolving intersections between AABB or other Triangle Nodes", 0, 4),
]

gltfExportTypes = [
    ("GLB", ".glb", "Exports a single file, with all data packed in binary form. Most efficient and portable, but more difficult to edit later", 0, 0),
    ("GLTF_SEPARATE", ".gltf + .bin + textures", "Exports multiple files, with separate JSON, binary and texture data. Easiest to edit later - Note that Tetra3D doesn't support this properly currently", 0, 1),
    ("GLTF_EMBEDDED", ".gltf", "Exports a single file, with all data packed in JSON. Less efficient than binary, but easier to edit later", 0, 2),
 ]

#class OBJECT_OT_tetra3dAddProp(bpy.types.Operator):
#    bl_idname = "object.tetra3daddprop"
#    bl_label = "Tetra3d Add Custom Property"
#    bl_options = {'REGISTER', 'UNDO'}
#    
#    def execute(self, context):
#        context.object.t3dCustomProperties__.append("PropName")
#        return {'FINISHED'}

class OBJECT_PT_tetra3d(bpy.types.Panel):
    bl_idname = "OBJECT_PT_tetra3d"
    bl_label = "Tetra3d Properties"
    bl_space_type = 'PROPERTIES'
    bl_region_type = 'WINDOW'
    bl_context = "object"

    def draw(self, context):
        row = self.layout.row()
        row.prop(context.object, "t3dVisible__")
        row = self.layout.row()
        row.prop(context.object, "t3dBoundsType__")
        row = self.layout.row()
        
        if context.object.t3dBoundsType__ == 'AABB':
            row.prop(context.object, "t3dAABBCustomEnabled__")
            if context.object.t3dAABBCustomEnabled__:
                row = self.layout.row()
                row.prop(context.object, "t3dAABBCustomSize__")
        elif context.object.t3dBoundsType__ == 'CAPSULE':
            row.prop(context.object, "t3dCapsuleCustomEnabled__")
            if context.object.t3dCapsuleCustomEnabled__:
                row = self.layout.row()
                row.prop(context.object, "t3dCapsuleCustomRadius__")
                row.prop(context.object, "t3dCapsuleCustomHeight__")
        elif context.object.t3dBoundsType__ == 'SPHERE':
            row.prop(context.object, "t3dSphereCustomEnabled__")
            if context.object.t3dSphereCustomEnabled__:
                row = self.layout.row()
                row.prop(context.object, "t3dSphereCustomRadius__")
                
#        row = self.layout.row()
#        row.operator("object.tetra3daddprop", text="Add Property")
        
# The idea behind "globalget and set" is that we're setting properties on the first scene (which must exist), and getting any property just returns the first one from that scene
def globalGet(propName):
    if propName in bpy.data.scenes[0]:
        return bpy.data.scenes[0][propName]

def globalSet(propName, value):
    bpy.data.scenes[0][propName] = value

class RENDER_PT_tetra3d(bpy.types.Panel):
    bl_idname = "RENDER_PT_tetra3d"
    bl_label = "Tetra3D Properties"
    bl_space_type = "PROPERTIES"
    bl_region_type = "WINDOW"
    bl_context = "render"
    
    def draw(self, context):
        row = self.layout.row()
        row.prop(context.scene, "t3dExportOnSave__")
        if globalGet("t3dExportOnSave__"):
            row = self.layout.row()
            row.prop(context.scene, "t3dExportFilepath__")
            
            row = self.layout.row()
            row.prop(context.scene, "t3dExportFormat__")
            
            box = self.layout.box()
            box.prop(context.scene, "t3dExportCameras__")
            box.prop(context.scene, "t3dExportLights__")


@persistent
def exportOnSave(dummy):
    
    if globalGet("t3dExportOnSave__"):
        scene = bpy.context.scene
        
        blendPath = bpy.context.blend_data.filepath
        if scene.t3dExportFilepath__ != "":
            blendPath = scene.t3dExportFilepath__
        
        if scene.t3dExportFormat__ == "GLB":
            ending = ".glb"
        elif scene.t3dExportFormat__ == "GLTF_SEPARATE" or scene.t3dExportFormat__ == "GLTF_EMBEDDED":
            ending = ".gltf"
        
        newPath = os.path.splitext(blendPath)[0] + ending

        for obj in bpy.data.objects:
            cloning = []
            if obj.instance_type == "COLLECTION":
                for o in obj.instance_collection.objects:
                    if o.parent == None:
                        cloning.append(o.name)
            if len(cloning) > 0:
                obj["t3dInstanceCollection__"] = cloning

        # We force on exporting of Extra values because otherwise, values from Blender would not be able to be exported.
        # export_apply=True to ensure modifiers are applied.
        bpy.ops.export_scene.gltf(
            filepath=newPath, 
            export_format=scene.t3dExportFormat__, 
            export_cameras=scene.t3dExportCameras__, 
            export_lights=scene.t3dExportLights__, 
            
            export_extras=True,
            export_yup=True,
            export_apply=True,
        )

        for obj in bpy.data.objects:
            if "t3dInstanceCollection__" in obj:
                del(obj["t3dInstanceCollection__"])

objectProps = {
    "t3dVisible__" : bpy.props.BoolProperty(name="Visible", description="Whether the object is visible or not when exported to Tetra3D", default=True),
    "t3dBoundsType__" : bpy.props.EnumProperty(items=boundsTypes, name="Bounds", description="What Bounding node type to create and parent to this object"),
    "t3dAABBCustomEnabled__" : bpy.props.BoolProperty(name="Custom AABB Size", description="If enabled, you can manually set the BoundingAABB node's size. If disabled, the AABB's size will be automatically determined by this object's mesh (if it is a mesh; otherwise, no BoundingAABB node will be generated)", default=False),
    "t3dAABBCustomSize__" : bpy.props.FloatVectorProperty(name="AABB Size", description="Width (X), height (Y), and depth (Z) of the BoundingAABB node that will be created", min=0.0, default=[2,2,2]),
    "t3dCapsuleCustomEnabled__" : bpy.props.BoolProperty(name="Custom Capsule Size", description="If enabled, you can manually set the BoundingCapsule node's size properties. If disabled, the Capsule's size will be automatically determined by this object's mesh (if it is a mesh; otherwise, no BoundingCapsule node will be generated)", default=False),
    "t3dCapsuleCustomRadius__" : bpy.props.FloatProperty(name="Capsule Radius", description="The radius of the BoundingCapsule node.", min=0.0, default=0.5),
    "t3dCapsuleCustomHeight__" : bpy.props.FloatProperty(name="Capsule Height", description="The height of the BoundingCapsule node.", min=0.0, default=2),
    "t3dSphereCustomEnabled__" : bpy.props.BoolProperty(name="Custom Sphere Size", description="If enabled, you can manually set the BoundingSphere node's radius. If disabled, the Sphere's size will be automatically determined by this object's mesh (if it is a mesh; otherwise, no BoundingSphere node will be generated)", default=False),
    "t3dSphereCustomRadius__" : bpy.props.FloatProperty(name="Sphere Radius", description="Radius of the BoundingSphere node that will be created", min=0.0, default=1),
}

def getExportOnSave(self):
    s = globalGet("t3dExportOnSave__")
    if s is None:
        s = False
    return s

def setExportOnSave(self, value):
    globalSet("t3dExportOnSave__", value)



def getExportFilepath(self):
    fp = globalGet("t3dExportFilepath__")
    if fp is None:
        fp = ""
    return fp

def setExportFilepath(self, value):
    globalSet("t3dExportFilepath__", value)



def getExportFormat(self):
    f = globalGet("t3dExportFormat__")
    if f is None:
        f = 0
    return f

def setExportFormat(self, value):
    globalSet("t3dExportFormat__", value)



def getExportCameras(self):
    c = globalGet("t3dExportCameras__")
    if c is None:
        c = True
    return c

def setExportCameras(self, value):
    globalSet("t3dExportCameras__", value)



def getExportLights(self):
    l = globalGet("t3dExportLights__")
    if l is None:
        l = True
    return l

def setExportLights(self, value):
    globalSet("t3dExportLights__", value)



def register():
    
    bpy.utils.register_class(OBJECT_PT_tetra3d)
    bpy.utils.register_class(RENDER_PT_tetra3d)
#    bpy.utils.register_class(OBJECT_OT_tetra3dAddProp)
    
    for propName, prop in objectProps.items():
        setattr(bpy.types.Object, propName, prop)

    bpy.types.Scene.t3dExportOnSave__ = bpy.props.BoolProperty(name="Export on Save", description="Whether the current file should export to GLTF on save or not", default=False, 
    get=getExportOnSave, set=setExportOnSave)
    
    bpy.types.Scene.t3dExportFilepath__ = bpy.props.StringProperty(name="Export Filepath", description="Filepath to export GLTF file. If left blank, it will export to the same directory as the blend file and will have the same filename; in this case, if the blend file has not been saved, nothing will happen", 
    default="", subtype="FILE_PATH", get=getExportFilepath, set=setExportFilepath)
    
    bpy.types.Scene.t3dExportFormat__ = bpy.props.EnumProperty(items=gltfExportTypes, name="Export Format", description="What format to export the file in", default="GLTF_EMBEDDED",
    get=getExportFormat, set=setExportFormat)
    
    bpy.types.Scene.t3dExportCameras__ = bpy.props.BoolProperty(name="Export Cameras", description="Whether Blender should export cameras to the GLTF file", default=True,
    get=getExportCameras, set=setExportCameras)

    bpy.types.Scene.t3dExportLights__ = bpy.props.BoolProperty(name="Export Lights", description="Whether Blender should export lights to the GLTF file", default=True,
    get=getExportLights, set=setExportLights)
    
    if not exportOnSave in bpy.app.handlers.save_post:
        bpy.app.handlers.save_post.append(exportOnSave)
    
def unregister():
    bpy.utils.unregister_class(OBJECT_PT_tetra3d)
    bpy.utils.unregister_class(RENDER_PT_tetra3d)
#    bpy.utils.unregister_class(OBJECT_OT_tetra3dAddProp)
    
    for propName, prop in objectProps.items():
        delattr(bpy.types.Object, propName)
    
    del bpy.types.Scene.t3dExportOnSave__
    del bpy.types.Scene.t3dExportFilepath__
    del bpy.types.Scene.t3dExportFormat__
    
    del bpy.types.Scene.t3dExportCameras__
    del bpy.types.Scene.t3dExportLights__
    # del bpy.types.Scene.t3dExportExtras__

    if exportOnSave in bpy.app.handlers.save_post:
        bpy.app.handlers.save_post.remove(exportOnSave)
    

if __name__ == "__main__":
    register()
